package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Chamado struct {
	ID    int64
	X, Y  int16
	Valor int16
	Prazo int16
	Ativo bool

	Cel       int32
	GradeProx int32
	GradeAnt  int32
	Reservado int

	ExpProx int
	ExpAnt  int

	HashProx int32
	HashAnt  int32
}

type Ambulancia struct {
	ID    int64
	X, Y  int16
	Livre bool
}

type Hospital struct {
	X, Y int16
}

const maxChamados = 1000000
const janelaPrazo = 2000
const tamCelula = 16
const valorMax = 100

var (
	leitor   *bufio.Scanner
	escritor *bufio.Writer

	hospitais []Hospital
	pool      *PoolChamados
	grade     *GradeEspacial
	tabelaIDs *TabelaIDs

	largura, altura int
	expCabecas      []int
)

// --- ARMAZENAMENTO DE CHAMADOS ---

// Chamados são armazenados em um array fixo (pool), referidos por índice.
// Índices de chamados removidos vão para uma pilha e são reaproveitados.
type PoolChamados struct {
	chamados []Chamado

	capacidade int
	proxIndice int // primeiro índice nunca usado
	livres     []int32
}

func NovoPool(capacidade int) *PoolChamados {
	return &PoolChamados{
		chamados:   make([]Chamado, capacidade),
		capacidade: capacidade,
		livres:     make([]int32, 0, capacidade),
	}
}

func (p *PoolChamados) Alloc() (int, bool) {
	if len(p.livres) > 0 {
		indice := int(p.livres[len(p.livres)-1])
		p.livres = p.livres[:len(p.livres)-1]
		return indice, true
	}
	if p.proxIndice >= p.capacidade {
		return 0, false
	}
	indice := p.proxIndice
	p.proxIndice++
	return indice, true
}

func (p *PoolChamados) Free(indice int) {
	if !p.chamados[indice].Ativo {
		return
	}

	p.chamados[indice] = Chamado{}
	p.livres = append(p.livres, int32(indice))
}

// --- GRADE ESPACIAL ---

type GradeEspacial struct {
	cols, linhas int
	cabecas      []int32 // primeiro chamado de cada célula, -1 se vazia
}

func NovaGrade(largura, altura int) *GradeEspacial {
	g := &GradeEspacial{
		cols:   (largura + tamCelula - 1) / tamCelula,
		linhas: (altura + tamCelula - 1) / tamCelula,
	}

	g.cabecas = make([]int32, g.cols*g.linhas)
	for i := range g.cabecas {
		g.cabecas[i] = -1
	}

	return g
}

func (g *GradeEspacial) indiceCelula(x, y int) int {
	return (y/tamCelula)*g.cols + x/tamCelula
}

func (g *GradeEspacial) Inserir(p *PoolChamados, indice int) {
	c := &p.chamados[indice]
	cel := g.indiceCelula(int(c.X), int(c.Y))

	c.Cel = int32(cel)
	c.GradeAnt = -1
	c.GradeProx = g.cabecas[cel]

	if c.GradeProx != -1 {
		p.chamados[c.GradeProx].GradeAnt = int32(indice)
	}

	g.cabecas[cel] = int32(indice)
}

// chamar antes do Free, que zera o chamado
func (g *GradeEspacial) Remover(p *PoolChamados, indice int) {
	c := &p.chamados[indice]

	if c.GradeAnt == -1 {
		g.cabecas[c.Cel] = c.GradeProx
	} else {
		p.chamados[c.GradeAnt].GradeProx = c.GradeProx
	}

	if c.GradeProx != -1 {
		p.chamados[c.GradeProx].GradeAnt = c.GradeAnt
	}

	c.GradeAnt = -1
	c.GradeProx = -1
}

// acrescenta a out os índices de chamados perto de (x, y), percorrendo celulas em anéis,
// até ter pelo menos k chamados (ou percorrer todas células).
func (g *GradeEspacial) Proximos(p *PoolChamados, x, y, k int, out []int) []int {
	cx0, cy0 := x/tamCelula, y/tamCelula
	maxR := max(g.cols, g.linhas)

	for r := 0; r <= maxR; r++ {
		for cy := cy0 - r; cy <= cy0+r; cy++ {
			if cy < 0 || cy >= g.linhas {
				continue
			}
			passo := 1
			if cy != cy0-r && cy != cy0+r {
				passo = 2 * r // r > 0 aqui, pois no anel 0 só existe uma linha
			}
			for cx := cx0 - r; cx <= cx0+r; cx += passo {
				if cx < 0 || cx >= g.cols {
					continue
				}
				for i := g.cabecas[cy*g.cols+cx]; i != -1; i = p.chamados[i].GradeProx {
					out = append(out, int(i))
				}
			}
		}
		if len(out) >= k {
			break
		}
	}
	return out
}

// --- LISTA DE EXPIRAÇÃO ---

func expInserir(p *PoolChamados, indice int) {
	c := &p.chamados[indice]
	b := c.Prazo % janelaPrazo
	c.ExpAnt = -1
	c.ExpProx = expCabecas[b]
	if c.ExpProx != -1 {
		p.chamados[c.ExpProx].ExpAnt = indice
	}
	expCabecas[b] = indice
}

func expRemover(p *PoolChamados, indice int) {
	c := &p.chamados[indice]
	if c.ExpAnt == -1 {
		expCabecas[c.Prazo%janelaPrazo] = c.ExpProx
	} else {
		p.chamados[c.ExpAnt].ExpProx = c.ExpProx
	}
	if c.ExpProx != -1 {
		p.chamados[c.ExpProx].ExpAnt = c.ExpAnt
	}
}

// -- TABELA HASH DE IDS ---

// ids com tabela hash, +fácil de localizar
type TabelaIDs struct {
	baldes []int32
}

func NovaTabelaIDs(capacidade int) *TabelaIDs {
	t := &TabelaIDs{
		baldes: make([]int32, capacidade*2+1),
	}

	for i := range t.baldes {
		t.baldes[i] = -1
	}

	return t
}

// função hash
func (t *TabelaIDs) hash(id int64) int {
	return int(uint(id) % uint(len(t.baldes)))
}

// inserção
func (t *TabelaIDs) Inserir(p *PoolChamados, indice int) {
	c := &p.chamados[indice]

	balde := t.hash(c.ID)

	c.HashAnt = -1
	c.HashProx = t.baldes[balde]

	if c.HashProx != -1 {
		p.chamados[c.HashProx].HashAnt = int32(indice)
	}

	t.baldes[balde] = int32(indice)
}

// busca por ID
func (t *TabelaIDs) Buscar(p *PoolChamados, id int64) (int, bool) {
	balde := t.hash(id)

	atual := t.baldes[balde]

	for atual != -1 {
		c := &p.chamados[atual]

		if c.Ativo && c.ID == id {
			return int(atual), true
		}

		atual = c.HashProx
	}

	return -1, false
}

// remoção
func (t *TabelaIDs) RemoverIndice(p *PoolChamados, indice int) {
	c := &p.chamados[indice]

	balde := t.hash(c.ID)

	if c.HashAnt == -1 {
		t.baldes[balde] = c.HashProx
	} else {
		p.chamados[c.HashAnt].HashProx = c.HashProx
	}

	if c.HashProx != -1 {
		p.chamados[c.HashProx].HashAnt = c.HashAnt
	}

	c.HashAnt = -1
	c.HashProx = -1
}

// lerCampos lê a próxima linha da entrada e a divide em campos.
// Devolve nil quando a entrada termina.
func lerCampos() []string {
	if !leitor.Scan() {
		return nil
	}
	return strings.Fields(leitor.Text())
}

func inteiro(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic("número inválido na entrada: " + s)
	}
	return v
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func distancia(x1, y1, x2, y2 int) int {
	return abs(x1-x2) + abs(y1-y2)
}

// direcao devolve o movimento que aproxima (x, y) de (alvoX, alvoY),
// andando primeiro na horizontal.
func direcao(x, y, alvoX, alvoY int) string {
	switch {
	case x < alvoX:
		return "L"
	case x > alvoX:
		return "O"
	case y < alvoY:
		return "S"
	case y > alvoY:
		return "N"
	}
	return "P"
}

// lerInicio lê a mensagem de início. Este bot só usa os hospitais; os
// bloqueios são lidos e descartados.
func lerInicio() {
	c := lerCampos() // INICIO largura altura turnos limite_turno limite_preparo equipe num_equipes
	largura = inteiro(c[1])
	altura = inteiro(c[2])
	h := inteiro(lerCampos()[1])
	for i := 0; i < h; i++ {
		c := lerCampos()
		hospitais = append(hospitais, Hospital{int16(inteiro(c[0])), int16(inteiro(c[1]))})
	}
	k := inteiro(lerCampos()[1])
	for i := 0; i < k; i++ {
		lerCampos() // id x y
	}
	b := inteiro(lerCampos()[1])
	for i := 0; i < b; i++ {
		lerCampos() // x1 y1 x2 y2
	}
	lerCampos() // FIM
}

func removerIndice(indice int) {
	tabelaIDs.RemoverIndice(pool, indice)
	grade.Remover(pool, indice)
	expRemover(pool, indice)
	pool.Free(indice)
}

func removerPorID(id int64) {
	indice, ok := tabelaIDs.Buscar(pool, id)
	if !ok {
		return
	}
	removerIndice(indice)
}

func removerExpirados(turno int) {
	if turno < 1 {
		return
	}
	b := (turno - 1) % janelaPrazo
	for i := expCabecas[b]; i != -1; {
		prox := pool.chamados[i].ExpProx
		if int(pool.chamados[i].Prazo) < turno {
			removerIndice(i)
		}
		i = prox
	}
}

func jogarTurno(turno int, ambulancias []Ambulancia) {
	type Par struct {
		ambIndice     int
		chamadoIndice int
		prioridade    float64
		distancia     int
	}

	// Controla usando slices, sem o mapa q o professo proibiu
	movimentos := make([]string, len(ambulancias))
	usadosAmbulancias := make([]bool, len(ambulancias))
	usadosChamados := make([]bool, pool.proxIndice)

	pares := make([]Par, 0)
	candidatos := make([]int, 0, 100)

	// Gera as combinações entre ambulâncias livres e chamados ativos.
	for ai, a := range ambulancias {
		if !a.Livre {
			continue
		}

		candidatos = grade.Proximos(pool, int(a.X), int(a.Y), 8, candidatos[:0])

		for _, ci := range candidatos {
			c := &pool.chamados[ci]

			dist := distancia(int(a.X), int(a.Y), int(c.X), int(c.Y))
			if !c.Ativo || int(c.Prazo) < turno+dist {
				continue
			}

			prioridade := float64(c.Valor) / float64(dist+1)

			pares = append(pares, Par{
				ambIndice:     ai,
				chamadoIndice: ci,
				prioridade:    prioridade,
				distancia:     dist,
			})
		}
	}

	// Ordena pela maior prioridade.
	// Em caso de empate, prefere o chamado mais próximo.
	sort.Slice(pares, func(i, j int) bool {
		if pares[i].prioridade == pares[j].prioridade {
			return pares[i].distancia < pares[j].distancia
		}

		return pares[i].prioridade > pares[j].prioridade
	})
	//questão que estava faltando, a saber quais ambulâncias estão ocupadas e tals.
	// Distribui os chamados sem repetir ambulâncias ou chamados.
	for _, p := range pares {
		if usadosAmbulancias[p.ambIndice] ||
			usadosChamados[p.chamadoIndice] {
			continue
		}

		a := ambulancias[p.ambIndice]
		c := pool.chamados[p.chamadoIndice]

		movimentos[p.ambIndice] = direcao(
			int(a.X), int(a.Y),
			int(c.X), int(c.Y),
		)

		usadosAmbulancias[p.ambIndice] = true
		usadosChamados[p.chamadoIndice] = true
	}

	// Ambulâncias ocupadas vão para o hospital mais próximo.
	for ai, a := range ambulancias {
		if !a.Livre {
			if len(hospitais) == 0 {
				movimentos[ai] = "P"
				continue
			}

			melhor := 0

			for i, h := range hospitais {
				if distancia(int(a.X), int(a.Y), int(h.X), int(h.Y)) <
					distancia(int(a.X), int(a.Y), int(hospitais[melhor].X), int(hospitais[melhor].Y)) {
					melhor = i
				}
			}

			h := hospitais[melhor]

			movimentos[ai] = direcao(
				int(a.X), int(a.Y),
				int(h.X), int(h.Y),
			)
		}

		// Se não encontrou chamado para uma ambulância livre, ela espera.
		if movimentos[ai] == "" {
			movimentos[ai] = "P"
		}
	}

	// Envia os movimentos na ordem original das ambulâncias.
	for ai, a := range ambulancias {
		fmt.Fprintln(escritor, a.ID, movimentos[ai])
	}

	fmt.Fprintln(escritor, "FIM")
	escritor.Flush()
}

func main() {
	leitor = bufio.NewScanner(os.Stdin)
	leitor.Buffer(make([]byte, 1<<16), 1<<20)
	escritor = bufio.NewWriter(os.Stdout)

	lerInicio()
	pool = NovoPool(maxChamados)
	grade = NovaGrade(largura, altura)
	expCabecas = make([]int, janelaPrazo)
	for i := range expCabecas {
		expCabecas[i] = -1
	}
	tabelaIDs = NovaTabelaIDs(maxChamados)

	fmt.Fprintln(escritor, "PRONTO")
	escritor.Flush()

	for {
		c := lerCampos() // TURNO t
		if c == nil {
			return // fim da partida
		}
		turno := inteiro(c[1])

		n := inteiro(lerCampos()[1]) // CHAMADOS n
		for i := 0; i < n; i++ {
			c := lerCampos()

			indice, ok := pool.Alloc()
			if !ok {
				continue // pool cheio
			}
			pool.chamados[indice] = Chamado{
				ID:       int64(inteiro(c[0])),
				X:        int16(inteiro(c[1])),
				Y:        int16(inteiro(c[2])),
				Valor:    int16(inteiro(c[3])),
				Prazo:    int16(inteiro(c[4])),
				Ativo:    true,
				HashProx: -1,
				HashAnt:  -1,
			}

			grade.Inserir(pool, indice)
			expInserir(pool, indice)
			tabelaIDs.Inserir(pool, indice)
		}

		r := inteiro(lerCampos()[1]) // RECOLHIDOS r
		for i := 0; i < r; i++ {
			removerPorID(int64(inteiro(lerCampos()[0])))
		}
		removerExpirados(turno)

		k := inteiro(lerCampos()[1]) // AMBULANCIAS k
		ambulancias := make([]Ambulancia, k)
		for i := 0; i < k; i++ {
			c := lerCampos()
			ambulancias[i] = Ambulancia{int64(inteiro(c[0])), int16(inteiro(c[1])), int16(inteiro(c[2])), c[3] == "LIVRE"}
		}

		e := inteiro(lerCampos()[1]) // PLACAR e
		for i := 0; i < e; i++ {
			lerCampos()
		}
		lerCampos() // FIM

		jogarTurno(turno, ambulancias)
	}
}

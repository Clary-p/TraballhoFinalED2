// Bot ingênuo: ponto de partida do trabalho prático.
//
// Este bot implementa o protocolo inteiro e joga de forma simples:
//
//   - guarda os chamados abertos num slice;
//   - a cada turno, cada ambulância livre procura, por varredura linear,
//     o chamado mais próximo e anda na direção dele;
//   - cada ambulância ocupada anda na direção do hospital mais próximo.
//
// Ele não verifica se o chamado ainda pode ser alcançado dentro do prazo,
// não impede que duas ambulâncias sigam para o mesmo chamado e ignora os
// bloqueios da etapa 2. No cenário de estresse, a varredura linear estoura
// o limite de tempo por turno.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const maxChamados = 1000000

type Chamado struct {
	ID    int
	X, Y  int
	Valor int
	Prazo int
	Ativo bool
	
    HashProx int
    HashAnt  int
}

type Ambulancia struct {
	ID    int
	X, Y  int
	Livre bool
}

type Hospital struct {
	X, Y int
}

// --- ARMAZENAMENTO DE CHAMADOS ---

// Chamados são armazenados em um array fixo (pool), referidos por índice.
// Índices de chamados removidos vão para uma pilha e são reaproveitados.
type PoolChamados struct {
	chamados []Chamado

	capacidade int
	proxIndice int // primeiro índice nunca usado
	livres     []int
}
//ids com tabela hash, +fácil de localizar
type TabelaIDs struct {
    baldes []int
}

func NovaTabelaIDs(capacidade int) *TabelaIDs {
    t := &TabelaIDs{
        baldes: make([]int, capacidade*2+1),
    }

    for i := range t.baldes {
        t.baldes[i] = -1
    }

    return t
}
//função hash
func (t *TabelaIDs) hash(id int) int {
    return int(uint(id) % uint(len(t.baldes)))
}
//inserção
func (t *TabelaIDs) Inserir(p *PoolChamados, indice int) {
    c := &p.chamados[indice]

    balde := t.hash(c.ID)

    c.HashAnt = -1
    c.HashProx = t.baldes[balde]

    if c.HashProx != -1 {
        p.chamados[c.HashProx].HashAnt = indice
    }

    t.baldes[balde] = indice
}

//busca por ID 
func (t *TabelaIDs) Buscar(p *PoolChamados, id int) (int, bool) {
    balde := t.hash(id)

    atual := t.baldes[balde]

    for atual != -1 {
        c := &p.chamados[atual]

        if c.Ativo && c.ID == id {
            return atual, true
        }

        atual = c.HashProx
    }

    return -1, false
}
//remoção
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

func NovoPool(capacidade int) *PoolChamados {
	return &PoolChamados{
		chamados:   make([]Chamado, capacidade),
		capacidade: capacidade,
		livres:     make([]int, 0, capacidade),
	}
}

func (p *PoolChamados) Alloc() (int, bool) {
	if len(p.livres) > 0 {
		indice := p.livres[len(p.livres)-1]
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
//modificado
func (p *PoolChamados) Free(indice int) {
    if !p.chamados[indice].Ativo {
        return
    }

    p.chamados[indice] = Chamado{}
    p.livres = append(p.livres, indice)
}

var (
    leitor    *bufio.Scanner
    escritor  *bufio.Writer

    hospitais []Hospital
    pool      *PoolChamados
    tabelaIDs *TabelaIDs
)

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
	lerCampos() // INICIO largura altura turnos limite_turno limite_preparo equipe num_equipes
	h := inteiro(lerCampos()[1])
	for i := 0; i < h; i++ {
		c := lerCampos()
		hospitais = append(hospitais, Hospital{inteiro(c[0]), inteiro(c[1])})
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

// MODIFICADO p/ hash
//removerPorID procura o chamado no slice e o remove
func removerPorID(id int) {
	indice, encontrado := tabelaIDs.Buscar(pool, id)

    if !encontrado {
        return
    }

    tabelaIDs.RemoverIndice(pool, indice)
    pool.Free(indice)
}
//MODIFICADO
// removerExpirados mantém só os chamados com prazo maior ou igual ao turno.
func removerExpirados(turno int) {
	 for i := 0; i < pool.proxIndice; i++ {
        c := &pool.chamados[i]
        if c.Ativo && c.Prazo < turno {
            tabelaIDs.RemoverIndice(pool, i)
            pool.Free(i)
        }
    }
}

func jogarTurno(turno int, ambulancias []Ambulancia) {
	for _, a := range ambulancias {
		mov := "P"
		if a.Livre {
			melhor := -1
			for i := 0; i < pool.proxIndice; i++ {
				c := &pool.chamados[i]
				if !c.Ativo {
					continue
				}
				if melhor == -1 || distancia(a.X, a.Y, c.X, c.Y) < distancia(a.X, a.Y, pool.chamados[melhor].X, pool.chamados[melhor].Y) {
					melhor = i
				}
			}
			if melhor != -1 {
				mov = direcao(a.X, a.Y, pool.chamados[melhor].X, pool.chamados[melhor].Y)
			}
		} else {
			melhor := 0
			for i, h := range hospitais {
				if distancia(a.X, a.Y, h.X, h.Y) < distancia(a.X, a.Y, hospitais[melhor].X, hospitais[melhor].Y) {
					melhor = i
				}
			}
			mov = direcao(a.X, a.Y, hospitais[melhor].X, hospitais[melhor].Y)
		}
		fmt.Fprintln(escritor, a.ID, mov)
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
    ID:       inteiro(c[0]),
    X:        inteiro(c[1]),
    Y:        inteiro(c[2]),
    Valor:    inteiro(c[3]),
    Prazo:    inteiro(c[4]),
    Ativo:    true,
    HashProx: -1,
    HashAnt:  -1,
}

tabelaIDs.Inserir(pool, indice)
		}

		r := inteiro(lerCampos()[1]) // RECOLHIDOS r
		for i := 0; i < r; i++ {
			removerPorID(inteiro(lerCampos()[0]))
		}
		removerExpirados(turno)

		k := inteiro(lerCampos()[1]) // AMBULANCIAS k
		ambulancias := make([]Ambulancia, k)
		for i := 0; i < k; i++ {
			c := lerCampos()
			ambulancias[i] = Ambulancia{inteiro(c[0]), inteiro(c[1]), inteiro(c[2]), c[3] == "LIVRE"}
		}

		e := inteiro(lerCampos()[1]) // PLACAR e
		for i := 0; i < e; i++ {
			lerCampos()
		}
		lerCampos() // FIM

		jogarTurno(turno, ambulancias)
	}
}

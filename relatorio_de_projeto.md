# Relatório de projeto: Clarissa Pietrangelo e Sabrina Dorigoni Pelentir

## Estrutura base: Pool de chamados

Os chamados são armazenados em um pool pré-alocado de posições (`[]Chamado`), utilizando índices fixos. Cada posição representa um chamado, e os espaços liberados são reutilizados por meio de uma pilha de índices livres. As operações Alloc e Free possuem custo O(1), pois utilizam acesso direto por índice e uma pilha de posições livres. E como os índices permanecem os mesmos, evita-se a necessidade de atualizar referências nas outras estruturas.

## Achar os chamados próximos de uma ambulância

Estrutura: Grade espacial uniforme, em células de 16 por 16. Cada célula guarda uma lista duplamente encadeada (por índices do pool) dos chamados abertos nela, e para cada ambulância livre, a função `Proximos` percorre as células em anéis ao redor de sua posição, até reunir pelo menos 8 candidatos (ou percorrer toda a grade). Cada candidato alcançável antes do prazo vira um par com prioridade:

`prioridade = valor / (distância + 1)`

Inicialmente, priorizar apenas os chamados mais próximos fazia com que muitos chamados de baixo valor fossem recolhidos, prejudicando a pontuação. Por isso, optamos por essa relação entre valor e distância, que ainda pode receber ajustes de pesos na etapa 2.

As combinações entre ambulâncias e chamados são armazenadas em outro slice e ordenadas por prioridade com `sort.Slice`.

**Custo por turno:**
- Inserir ou remover um chamado da grade: O(1).
- Percorrer as células em anéis: O(R²), sendo R o número de anéis visitados.
- Percorrer os chamados presentes nas células visitadas: O(M), sendo M o número de candidatos encontrados.
- Gerar as combinações válidas: O(K).
- Ordenar as combinações: O(K log K).
- Distribuir os chamados: O(K).

Portanto, o custo total da escolha é O(A × (R² + M) + K log K), sendo A o número de ambulâncias livres. Esse custo não depende de uma varredura completa do pool.

**Alternativas descartadas:** O spatial hash utiliza uma ideia semelhante à grade espacial, mas é mais interessante para espaços ilimitados ou esparsos. Como nosso mapa possui dimensões limitadas, optamos pela grade uniforme, com acesso direto às células. Também consideramos quadtree e k-d tree, que permitem buscas espaciais, mas adicionariam complexidade à implementação e à manutenção da estrutura diante das inserções e remoções frequentes. Além disso, uma busca sequencial que escolhesse apenas o chamado mais próximo, sem considerar o valor, poderia priorizar muitos chamados de baixo valor.

## Descartar os chamados expirados

Estrutura: Pool de chamados `[]Chamado` associado à tabela hash de IDs e a uma roda de prazos com 2.000 baldes. Cada chamado é inserido no balde correspondente a `Prazo % janelaPrazo`, utilizando uma lista duplamente encadeada representada por índices do próprio pool.

A função `removerExpirados` consulta somente o balde correspondente ao prazo do turno anterior ((turno - 1) % janelaPrazo). Como o prazo relativo máximo dos chamados é de 1.000 turnos, menor que a janela de 2.000, cada balde contém chamados de um mesmo prazo enquanto eles estiverem ativos. Desse modo, não é necessário percorrer todos os índices utilizados no pool.

Quando um chamado expira, sua referência é removida da tabela hash e da grade espacial, sua posição é removida da lista de expiração e, por fim, liberada para reutilização no pool.

**Custo por turno:**
- Acessar o balde correspondente ao turno: O(1).
- Percorrer os chamados que expiram naquele turno: O(E).
- Remover cada chamado das estruturas: O(1) por chamado.
- Liberar cada posição no pool: O(1).

Portanto, a expiração custa O(E) por turno. Na prática, isso dá O(1) por chamado, pois cada um é verificado apenas uma vez.

**Alternativas descartadas:** Uma possibilidade seria utilizar um heap de prazos, com custo O(log N) por inserção e remoção, mas seria necessário implementar e manter essa estrutura manualmente. Optamos pela roda de prazos por permitir acessar diretamente os chamados que expiram em cada turno.

## Remover um chamado que outra equipe recolheu, sabendo apenas seu identificador

Estrutura: Tabela hash com encadeamento separado, utilizando listas duplamente encadeadas representadas por índices dentro do próprio pool de chamados.

A tabela tem um vetor de baldes. Cada chamado armazena:
- `HashProx`: índice do próximo chamado no mesmo balde.
- `HashAnt`: índice do chamado anterior no mesmo balde.

Quando recebemos o identificador de um chamado, calculamos seu balde por meio da função hash e percorremos somente os elementos daquele encadeamento.

Depois de encontrar o chamado, conseguimos removê-lo da tabela utilizando seus índices anterior e próximo, sem precisar percorrer toda a estrutura novamente. Também removemos o chamado da grade espacial e da lista de expiração. Por fim, liberamos sua posição no pool.

**Custo por turno:**
- Calcular o hash do identificador: O(1).
- Buscar o chamado na tabela: O(1) esperado, considerando uma distribuição adequada dos IDs entre os baldes; O(N) no pior caso, se houver muitas colisões.
- Remover o chamado do encadeamento, conhecendo seu índice: O(1).
- Liberar a posição no pool: O(1).

**Alternativas descartadas:** Percorrer sequencialmente todos os chamados procurando o identificador. Essa alternativa teria custo O(P) por busca. Também descartamos deslocar os elementos do slice após uma remoção, porque isso mudaria os índices utilizados pelas outras estruturas e exigiria atualizar suas referências. Por isso, optamos por manter posições fixas e reutilizar os espaços liberados.Além também de descartar lápides por causa do alto número de chamados em fases mais avançadas.

## Evitar que duas ambulâncias da equipe sigam para o mesmo chamado

Estrutura: Dois slices de valores booleanos `[]bool`, um para controlar as ambulâncias já utilizadas e outro para controlar os chamados já atribuídos. Durante a distribuição dos chamados, verificamos se a ambulância e o chamado da combinação já foram utilizados. Se qualquer um dos dois já tiver sido escolhido, ignoramos a combinação. Basicamente se decidimos a combinação X e Y, mas X OU Y já foi escolhido, descarta

Quando uma combinação é aceita, marcamos os dois índices como utilizados.

**Custo por turno:**
- Inicializar os slices de controle: O(A + P).
- Consultar se uma ambulância ou chamado já foi utilizado: O(1).
- Marcar uma ambulância e um chamado como utilizados: O(1).
- Percorrer as combinações ordenadas: O(K).

O custo da distribuição, depois da ordenação, é O(K), além da inicialização dos vetores.

**Alternativa descartada:** Foi considerado endereçamento aberto também, mas, como mencionado em outros problemas acima, optamos pelos slices porque os índices das ambulâncias e dos chamados já estão disponíveis, permitindo acesso direto por posição, sem necessidade de uma estrutura associativa.

## Encontrar o hospital mais próximo de uma ambulância ocupada

Estrutura: A lista de hospitais é armazenada em um slice. Para cada ambulância ocupada, percorremos os hospitais e calculamos a distância de Manhattan até cada um deles, guardamos o índice do hospital com a menor distância e, ao final, definimos a direção do movimento da ambulância até ele.

**Custo por turno:**
- Percorrer os hospitais para cada ambulância ocupada: O(H).
- Calcular e comparar as distâncias: O(1) por hospital.
- Definir a direção do movimento: O(1).

Portanto, o custo total é O(B × H), sendo B o número de ambulâncias ocupadas e H o número de hospitais.

**Alternativa descartada:** Poderíamos pré-calcular, durante o preparo, o hospital mais próximo de cada célula do mapa e armazenar o resultado em uma estrutura de acesso direto, permitindo consultas em O(1). Entretanto, como temos no máximo 20 hospitais, não daria muita diferença de processamento (na etapa atual), então uma varredura sequencial serve.

---

## Observações

- A: ambulâncias livres.
- B: ambulâncias ocupadas.
- P: maior índice utilizado no pool de chamados (`pool.proxIndice`).
- N: chamados ativos.
- K: combinações entre ambulâncias e chamados.
- E: chamados expirados no turno.
- M: candidatos encontrados nas células visitadas.
- R: anéis percorridos pela grade espacial.
- H: hospitais.

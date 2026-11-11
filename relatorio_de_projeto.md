# Relatório de projeto: Clarissa Pietrangelo e Sabrina Dorigoni Pelentir

## Achar os chamados próximos de uma ambulância

Estrutura: Pool de chamados implementado com um slice []Chamado, utilizando índices fixos. Cada posição representa um chamado, e os espaços liberados são reutilizados por meio de uma pilha de índices livres.

Na função "jogarTurno", percorre os chamados armazenados no pool, vendo quais estão ativos e dentro do prazo. Para cada ambulância livre, calcula-se uma razão entre a distância de Manhattan (que tá lá no enunciado do trabalho) até cada chamado válido e sua prioridade -:

"prioridade = valor / (distância + 1)"

- razão sujeita a modificação de pesos pra etapa 2, tem a questão de balanceamento entre valor do chamado e distãncia, havia um problema no começo de prioridade completa dos chamados mais próximos, então acabamos ficando com muitos chamados recolhidos, mas eram de menor valor, então ainda perdemos, aí surgiu essa razão.

As combinações entre ambulâncias e chamados são armazenadas em outro slice e ordenadas por prioridade com `sort.Slice`.

Custo por turno:
- Percorrer os chamados para cada ambulância livre: O(A × P)- A = o número de ambulâncias livres e P o maior índice já utilizado no pool.
- Gerar as combinações válidas: O(K), K= o número de combinações entre ambulâncias e chamados ativos. (O(n) para n = K)
- Ordenar as combinações: O(K log K).
- Distribuir os chamados: O(K).

Portanto, o custo total da escolha é O(A × P + K log K).

Alternativa descartada: Uma busca sequencial que escolhesse apenas o chamado mais próximo, sem considerar o valor. Ela era mais simples, mas poderia priorizar muitos chamados de baixo valor, prejudicando a pontuação. Também seria possível implementar uma estrutura espacial para localizar somente os chamados próximos, mas isso aumentaria a complexidade da implementação. Atualmente, ainda fazemos a varredura do pool.

## Descartar os chamados expirados

Estrutura: Pool de chamados []Chamado associado à tabela hash de IDs. Quando um chamado expira, ele é removido da tabela hash e sua posição no pool é liberada para reutilização.

A função removerExpirados percorre os índices utilizados do pool, verifica se o chamado está ativo e se seu prazo já terminou. Se estiver expirado, remove sua referência da tabela hash e libera sua posição, porém, a questão de varredura deles entra em outra parte (sabrina favor completar)

Custo por turno:
- Percorrer os índices utilizados do pool: O(P).
- Verificar se cada chamado está ativo e expirado: O(1) por posição.
- Remover um chamado da tabela hash, conhecendo seu índice: O(1).
- Liberar a posição no pool: O(1) amortizado.

O custo total da identificação e remoção dos expirados é O(P).

Alternativa considerada: lápides, foi considerado mas descardado porque nos níveis mais avançados poderia sobrecarregar o bot por conta de excesso de lápides (utiliza-se uma para indicar que o elemento foi removido), prejudicando o desempenho

## Remover um chamado que outra equipe recolheu, sabendo apenas seu identificador

Estrutura: Tabela hash com encadeamento separado, utilizando listas duplamente encadeadas representadas por índices dentro do próprio pool de chamados.

A tabela tem um vetor de baldes. Cada chamado armazena:
- HashProx: índice do próximo chamado no mesmo balde.
- HashAnt: índice do chamado anterior no mesmo balde.

Quando recebemos o identificador de um chamado, calculamos seu balde por meio da função hash e percorremos somente os elementos daquele encadeamento.

Depois de encontrar o chamado, conseguimos removê-lo da tabela utilizando seus índices anterior e próximo, sem precisar percorrer toda a estrutura novamente. Por fim, liberamos sua posição no pool.

Custo por turno:
- Calcular o hash do identificador: O(1).
- Buscar o chamado na tabela: O(1) esperado, considerando uma distribuição adequada dos IDs entre os baldes; O(N) no pior caso, se houver muitas colisões.
- Remover o chamado do encadeamento, conhecendo seu índice: O(1).
- Liberar a posição no pool: O(1) amortizado.

Alternativa descartada: Percorrer sequencialmente todos os chamados procurando o identificador. Essa alternativa teria custo O(P) por busca. Também descartamos deslocar os elementos do slice após uma remoção, porque isso mudaria os índices utilizados pelas outras estruturas e exigiria atualizar suas referências. Por isso, optamos por manter posições fixas e reutilizar os espaços liberados.Além também de descartar lápides por causa do alto número de chamados em fases mais avançadas, como na questão anterior.

## Evitar que duas ambulâncias da equipe sigam para o mesmo chamado

Estrutura: Dois slices de valores booleanos []bool, um para controlar as ambulâncias já utilizadas e outro para controlar os chamados já atribuídos. 
Durante a distribuição dos chamados, verificamos se a ambulância e o chamado da combinação já foram utilizados. Se qualquer um dos dois já tiver sido escolhido, ignoramos a combinação. Basicamente se decidimos a combinação X e Y, mas X OU Y já foi escolhido, descarta

Quando uma combinação é aceita, marcamos os dois índices como utilizados.

Custo por turno:
- Inicializar os slices de controle: O(A + P).
- Consultar se uma ambulância ou chamado já foi utilizado: O(1).
- Marcar uma ambulância e um chamado como utilizados: O(1).
- Percorrer as combinações ordenadas: O(K).

O custo da distribuição, depois da ordenação, é O(K), além da inicialização dos vetores.

Alternativa descartada: Foi considerado endereçamento aberto tmb, mas como mencionado em outros problemas acima, fica muito pesado nos níveis mais altos,optamos pelos slices porque os índices das ambulâncias e dos chamados já estão disponíveis, permitindo acesso direto por posição, sem necessidade de uma estrutura associativa. 
---

## Obs:

- A:ambulâncias livres.
- P: maior índice utilizado no pool de chamados (`pool.proxIndice`).
- N:chamados ativos.
- K: combinações entre ambulâncias e chamados.
- H: hospitais.


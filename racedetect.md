## Introduzione

Uno dei punti di forza di Go è la sua ricca toolchain, che integra moltissime funzionalità
quali un sistema di build, un package manager, un driver di testsuite, un profiler, e
molto altro ancora. Avere una toolchain così ricca e mantenuta assieme al linguaggio stesso
permette all'intero ecosistema di librerie e applicazioni Go di avere un comportamento
predefinito e standard, per cui se modificate una base di codice scritta da un collega o
scaricata da Internet, non c'è bisogno di documentarsi su come scrivere un test o eseguire
la testsuite, perché tutti i programmi in Go usano la stessa struttura per la scrittura
dei test.

Il race detector è una delle funzionalità più avanzate presenti nella toolchain di Go,
che (come vedremo) è utilissimo per debuggare problemi di concorrenza e locking.
Come probabilmente già sapete, Go è conosciuto per il potente supporto alla programmazione
concorrente (basato sulla scrittura di codice in stile "bloccante" che diventa
automaticamente asincrono grazie alle coroutine gestite dal runtime), e di conseguenza
molti programmi scritti in Go tendono a beneficiare di questo supporto, eseguendo decine
o anche migliaia di goroutine. Il race detector è pensato per facilitare il debugging
del codice Go.

## Problemi di concorrenza








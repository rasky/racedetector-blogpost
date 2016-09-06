## Introduzione

Uno dei punti di forza di Go è la sua **ricca toolchain**, che integra moltissime funzionalità
quali un sistema di build, un package manager, un sistema per test automatizzati, un profiler, e
molto altro ancora.

Il **race detector** è una delle funzionalità più avanzate presenti nella toolchain di Go,
che (come vedremo) è utilissimo per debuggare problemi di concorrenza e locking.
Come probabilmente già sapete, Go è conosciuto per il [potente supporto alla programmazione](https://www.develer.com/blog/concurrency-in-go)
concorrente (basato sulla scrittura di codice in stile "bloccante" che diventa
automaticamente asincrono grazie alle coroutine gestite dal runtime), e di conseguenza
molti programmi scritti in Go tendono a beneficiare di questo supporto, eseguendo decine
o anche **migliaia di goroutine**. Il race detector è pensato per facilitare il debugging
di software concorrente, aiutandovi ad **identificare le race condition** che possono
avvenire come risultato di tipici bug quali la mancanza di un mutex.

## La concorrenza e lo stato condiviso

Go utilizza un **modello di memoria condiviso** per le goroutine, esattamente come in C++
o Python per i thread; ciò vuol dire che ogni goroutine ha accesso a tutta la memoria
del processo all'interno del quale gira,
ed è quindi necessaria qualche cautela nell'accedere e modificare lo stato condiviso.
Tipicamente, questo vuol dire usare **primitive di sincronizzazione** come semafori o mutex,
oppure usufruire delle istruzioni speciali di accesso atomico alla memoria disponibili
nella maggior parte dei processori.

Dimenticarsi di effettuare un lock nel punto giusto è una fonte di bug tra i più
insidiosi. Il programma infatti può apparentemente funzionare in modo corretto durante
lo sviluppo o nelle prime prove in produzione; ma poi rischia di avere
**comportamenti imprevedibili** e difficilmente riproducibili, causando degli [heisenbug](https://it.wikipedia.org/wiki/Heisenbug)
fastidiosissimi. Purtroppo, nella stragrande maggioranza dei casi, gli sviluppatori
non hanno strumenti a disposizione che li aiutino ad accorgersi di questi problemi, e la correttezza
del programma è quindi affidata alla bravura e all'attenzione di chi scrive il codice
e di chi lo modifica. E vi posso assicurare che ho visto bug del genere nel codice
scritto da programmatori molto, molto esperti!

Il problema è sicuramente insidioso di per sé, ed in un certo senso è anche acutizzato
da un linguaggio con un potente e veloce supporto alla concorrenza come Go. In Go
è così **facile ed efficiente scrivere codice concorrente**, che è normale abusarne molto
più che in altri linguaggi, e questo rischia di innescare una spirale negativa che
allontana sempre di più la correttezza del codice... se non fosse che gli autori
di Go hanno pensato di aiutare i programmatori fornendo un potentissimo race detector
a pochi tasti di distanza.


## Esempio: contatore condiviso

Prendiamo come esempio un semplice programma Go [`counter.go`](counter.go) che espone un server TCP
e conta il numero di client che si collegano. Il codice che riporto è stato scritto
in modo un po' più ricco del minimo indispensabile, perché voglio mostrare un caso
realistico: ho implementato quindi una classe `Server` con un metodo bloccante
`Serve`, e un metodo `handleClient` che viene chiamato, in una goroutine separata, per ogni client che si connette.

```go
// counter.go: simple race detection example
package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

type Server struct {
	conn       net.Listener
	numClients int
}

// NewServer creates a new Server that will listen on the specified proto/addr combo.
// See net.Dial for documentation on proto and addr.
func NewServer(proto, addr string) (*Server, error) {
	conn, err := net.Listen(proto, addr)
	if err != nil {
		return nil, err
	}

	return &Server{conn: conn}, nil
}

// Serve makes Server listen for incoming connection, and spawn a goroutine calling handleClient
// for each new connection.
func (srv *Server) Serve() {
	for {
		conn, err := srv.conn.Accept()
		if err != nil {
			log.Print(err)
			return
		}

		srv.numClients += 1
		go srv.handleClient(conn)
	}
}

// handleClient manages the communication with a single client.
// In this example, we just send a predefined message and close the door
func (srv *Server) handleClient(conn net.Conn) {
	io.WriteString(conn, fmt.Sprintf("Ciao, sei il client #%d che si connette a me\n", srv.numClients))
	conn.Close()
}

func main() {
	srv, err := NewServer("tcp", "localhost:2380")
	if err != nil {
		log.Fatal(err)
	}

	srv.Serve()
}
```

Per eseguire e provare questo programma, in un terminale lanciamo `go run counter.go`, mentre in un altro
proviamo ad eseguire più volte `telent localhost 2380`. Dovremmo vedere qualcosa di questo genere:

```
$ telnet localhost 2380
Trying ::1...
Connected to localhost.
Escape character is '^]'.
Ciao, sei il client #1 che si connette a me
Connection closed by foreign host.

$ telnet localhost 2380
Trying ::1...
Connected to localhost.
Escape character is '^]'.
Ciao, sei il client #2 che si connette a me
Connection closed by foreign host.

$ telnet localhost 2380
Trying ::1...
Connected to localhost.
Escape character is '^]'.
Ciao, sei il client #3 che si connette a me
Connection closed by foreign host.
```

Come vedete, apparantemente il programma funziona correttamente. Ma è davvero così? A questo punto,
proviamo ad eseguire il programma attivando il race detector: è sufficiente passare l'opzione
`-race` a `go run`: quindi `go run -race counter.go`. Se ora proviamo a connetterci con `telnet`,
la prima volta andrà tutto bene, ma la seconda volta vedremo improvvisamente questo output
apparire nel terminale in cui il server è in esecuzione:

```
$ go run -race counter.go
==================
WARNING: DATA RACE
Write at 0x00c420086190 by main goroutine:
  main.(*Server).Serve()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:37 +0xae
  main.main()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:55 +0x86

Previous read at 0x00c420086190 by goroutine 7:
  runtime.convT2E()
      /usr/local/Cellar/go/1.7/libexec/src/runtime/iface.go:155 +0x0
  main.(*Server).handleClient()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:45 +0x69

Goroutine 7 (finished) created at:
  main.(*Server).Serve()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:38 +0xf0
  main.main()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:55 +0x86
==================
```

Come vedete il race detector ha individuato una data race. Si è accorto che due
goroutine hanno effettuato **una scrittura e una lettura alla stessa locazione di memoria**
(in questo caso: `0x00c420086190`) senza che ci fosse tra loro una sincronizzazione
esplicita; ci mostra lo stack-trace di ciascuna lettura/scrittura, l'ID
e lo stack-trace di creazione di ciasuna goroutine.

In questo caso, parafrasando quanto scritto sopra, si può dire che:

 * La goroutine "main" (quella di avvio del programma) ha effettuato una scrittura
   alla riga `counter.go:37`
 * La goroutine 7 ha effettuato una lettura alla stessa locazione di memoria da dentro
   il runtime di Go, ma lo stack trace ci indica che la chiamata è stata comunque generata
   dal nostro codice alla riga `counter.go:45`
 * La goroutine 7 è stata creata alla posizione `counter.go:38`.

Se guardiamo quindi il codice, vediamo che il race detector ci avverte che l'incremento
della variabile `numClients` e la lettura che ne viene fatta per stampare il valore sono
in potenziale conflitto tra loro. Infatti, non esistono sincronizzazioni tra questi due
statement.

E' importante notare che il race detector si è accorto del problema **nonostante le nostre
connessioni telnet fossero completamente sequenziali** e non parallele. In altre parole,
il race detector è in grado di identificare problemi di concorrenza **senza che questi
si verifichino davvero**. Non è quindi necessario affidarsi ai proverbiali santi e sperare
che il problema si verifichi mentre il race detector è attivo: è sufficiente eseguire il
codice da testare in una condizione semi-realistica e il race detector farà comunque il
suo lavoro.


## Come risolvere una data race

Come risolvere il problema identificato dal race detector? Un primo approccio può essere quello di
**introdurre un mutex** per sincronizzare tra loro gli accessi. Questo è un estratto di
[`counter_mutex.go`](mutex/counter_mutex.go) che mostra come viene introdotto:

```go
[...]

type Server struct {
	conn          net.Listener
	numClientLock sync.Mutex
	numClients    int
}

[...]

		srv.numClientLock.Lock()
		srv.numClients += 1
		srv.numClientLock.Unlock()

[...]

func (srv *Server) handleClient(conn net.Conn) {
	srv.numClientLock.Lock()
	nc := srv.numClients
	srv.numClientLock.Unlock()
	io.WriteString(conn, fmt.Sprintf("Ciao, sei il client #%d che si connette a me\n", nc))

[...]
```

Se provate ad eseguire ora il programma tramite `go run -race counter_mutex.go` e provate
ad effettuare connessioni successive, vedrete che il race detector non si lamenterà più
del problema. Per maggiori informazioni sull'uso dei mutex, potete leggere la documentazione
di [`sync.Mutex`](https://golang.org/pkg/sync/#Mutex). Ci sono anche altre primitive di sincronizzazione
a disposizione, come per esempio [`sync.RWMutex`](https://golang.org/pkg/sync/#RWMutex) o
[`sync.Once`](https://golang.org/pkg/sync/#Once).

Un piccolo suggerimento su questo argomento: nella scrittura del codice, è sempre bene tenere i lock
per il minor tempo possibile, e infatti ho preferito isolare la lettura dello stato condiviso in uno statement
separato, evitando di effettuare il lock intorno alla `io.WriteString`, che lo avrebbe
mantenuto bloccato anche durante l'intero I/O di rete.

Un altro approccio possibile in questo specifico caso, trattandosi di una concorrenza su una semplice
variabile di tipo integer, è quello di utilizzare le istruzioni atomiche del processore. Questo l'estratto
di [`counter_atomic.go`](atomic/counter_atomic.go) che mostra come fare:

```go
[...]

type Server struct {
	conn       net.Listener
	numClients int64
}

[...]

		atomic.AddInt64(&srv.numClients, 1)

[...]

func (srv *Server) handleClient(conn net.Conn) {
	nc := atomic.LoadInt64(&srv.numClients)
	io.WriteString(conn, fmt.Sprintf("Ciao, sei il client #%d che si connette a me\n", nc))

[...]
```

In questo caso, abbiamo utilizzato la funzione [`atomic.AddInt64`](https://golang.org/pkg/sync/atomic/#AddInt64)
per effettuare un incremento atomico, mentre la lettura atomica è demandata a
[`atomic.LoadInt64`](https://golang.org/pkg/sync/atomic/#LoadInt64). Gli accessi atomici sono un'alternativa
interessante ai mutex perché sono molto più veloci anche perché non causano context-switch. Si tratta
però di primitive un po' complesse da usare, per cui è meglio utilizzarle solo laddove si misurino
effettivi problemi di performance (condizione spesso rara); per maggiori informazioni, potete leggere
la documentazione del package [`sync/atomic`](https://golang.org/pkg/sync/atomic/).

Interessante anche notare la potenza del race detector in questo caso: se proviamo a lasciare la
`atomic.AddInt64` ma togliere la `atomic.LoadInt64`, viene comunque segnalata una data race. Questo
può sembrare ovvio inizialmente, ma in realtà **non lo è affatto**: infatti, su x86-64, mentre la
`atomic.AddInt64` è implementata tramite una istruzione assembly speciale (`LOCK XADDQ`), la
`atomic.LoadInt64` non è altro che un normale accesso alla memoria, perché l'architettura x86-64
garantisce che le lettura a 64-bit dalla memoria siano già atomiche. Di conseguenza, il race
detector non solo ci sta segnalando una potenziale data race, ma addirittura una data race
che si può verificare **solo su architetture diverse da quella in cui viene eseguito**, come
per esempio ARM32, in cui la lettura di una variabile a 64-bit deve necessariamente avvenire
con due diversi accessi alla memoria, e quindi in modo non atomico.


## Integrazione con la testsuite

Testare a mano un server TCP può essere un compito alquanto tedioso, e, si sa, i programmatori sono
tra i professionisti più pigri su questo pianeta. E' quindi sempre consigliato avere a disposizione
una testsuite automatizzata, e Go ci aiuta fornendoci delle librerie e un comodo supporto integrato
nella toolchain. Vediamo come scrivere un semplice test del nostro server: questo è il contenuto
di [`counter_test.go`](counter_test.go).

```go
// counter_test.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	srv, err := NewServer("tcp", "localhost:2380")
	if err != nil {
		t.Fatal(err)
	}

	go srv.Serve()
	defer srv.Close()

	for i := 0; i < 5; i++ {
		c, err := net.Dial("tcp", "localhost:2380")
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()

		line, err := bufio.NewReader(c).ReadString('\n')
		if err != nil || !strings.Contains(line, fmt.Sprintf("#%d ", i+1)) {
			t.Errorf("invalid text received: %q (err:%v)", line, err)
			return
		}
	}
}
```

Per eseguire questo test, è sufficente lanciare `go test` nella directory del progetto. Ovviamente,
la versione con data race sembrerà funzionare perfettamente:

```
$ go test
PASS
ok     	_/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d 	0.016s
```

Ma anche qui, è sufficiente aggiungere `-race` per accorgersi del problema:

```
$ go test -race
==================
WARNING: DATA RACE
Write at 0x00c420082190 by goroutine 8:
  _/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d.(*Server).Serve()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:37 +0xae

Previous read at 0x00c420082190 by goroutine 10:
  runtime.convT2E()
      /usr/local/Cellar/go/1.7/libexec/src/runtime/iface.go:155 +0x0
  _/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d.(*Server).handleClient()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:49 +0x69

Goroutine 8 (running) created at:
  _/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d.TestServer()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter_test.go:18 +0xe4
  testing.tRunner()
      /usr/local/Cellar/go/1.7/libexec/src/testing/testing.go:610 +0xc9

Goroutine 10 (finished) created at:
  _/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d.(*Server).Serve()
      /Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d/counter.go:38 +0xf0
==================
PASS
2016/08/20 17:30:32 accept tcp 127.0.0.1:2380: use of closed network connection
Found 1 data race(s)
exit status 66
FAIL   	_/Users/rasky/Sources/develer/e4daef8b5f9770c38439bf2310bc7b5d 	1.029s
```

Come potete vedere, il test in sé è passato (`PASS`) perché non è stata invocata una
funzione della libreria di test per marcare un errore (come per esempio `t.Error`), ma
l'intera testsuite viene marcata come `FAIL` perché è stata trovata una data race
durante l'esecuzione. Anche se quindi la data race non ha causato di per sé un
malfunzionamento tale da far fallire il test, Go ci suggerisce che ci sono comunque
problemi importanti rilevati dalla testsuite da sistemare.

E' buona norma utilizzare un sistema di continuous integration come [Travis CI](http://travis-ci.org) o
[Circle CI](http://www.circleci.com) per eseguire la testsuite su ogni commit effettuato
dal team. In questo caso, per i software Go, conviene che il CI esegua sempre la testsuite
tramite `go test -race`, in modo da accorgersi quanto prima possibile di problemi di
concorrenza.

## Conclusione

Il race detector è quindi un'arma molto importante nell'arsenale di ogni programmatore,
e i programmatori Go possono dormire sogni tranquilli sapendo di averne uno così potente
perfettamente integrato nella toolchain standard e a disposizione in ogni momento.

Il race detector è disponibile ad oggi solo su architetture a 64-bit; se quindi utilizzate
Go per fare compilazione su sistemi embedded a 32-bit quali quelli ARM, non potrete purtroppo
eseguire il race detector nativamente sul dispositivo target. In questo caso, consiglio
sempre di mantenere fin dall'inizio dello sviluppo la possibilità di eseguire il programma
(o almeno una parte significativa dello stesso) sul vostro sistema di sviluppo, in modo da
poter beneficiare di questa e numerosi altre funzionalità senza dover ogni volta passare
dall'esecuzione sul target.

Se volete avere più informazioni sul race detector, potete continuare la lettura
nella [documentazione ufficiale](https://golang.org/doc/articles/race_detector.html).


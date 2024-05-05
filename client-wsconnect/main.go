package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/net/websocket"
)

const (
	ccURL      string        = "ws://localhost:80/ws"
	ccQtd      int64         = 120
	ccInterval time.Duration = 60 * time.Second
	ccSample   int64         = 10
	ccSecret   string        = "bSh4cGORqEt8bAWYe3Dk"
)

func main() {

	// Carregar parâmetros da linha de comando
	qtdPtr := flag.Int64("q", ccQtd, "quantidade de clientes")
	intervalPtr := flag.String("i", "", "rampa de intervalo de conexão para os clientes no formato <número><unid.medida>, por exemplo, 5m (default 1m)")
	urlPtr := flag.String("u", ccURL, "endpoint de acesso ao ws, por exemplo, ws://localhost:80/ws")
	initialPtr := flag.Int64("id", 1, "identificador inicial do cliente")
	flag.Parse()

	var err error
	var qtd int64 = *qtdPtr
	var uri string = *urlPtr
	var initialID = *initialPtr
	var interval time.Duration = ccInterval

	if *intervalPtr != "" {
		if interval, err = time.ParseDuration(*intervalPtr); err != nil {
			log.Fatalln("Formato do intervalo inválido, precisa ser <número><unid.medida>, por exemplo, 5m (cinco minutos)", err)
		}
	}

	ini := time.Now()
	nextSample := ccSample
	wait := time.Duration(interval.Abs().Microseconds()/int64(qtd)) * time.Microsecond

	log.Println(" - WSCONNECT - Iniciando rampa de conexão de sockets em ", ini.Format(time.DateTime))
	fmt.Println(fmt.Sprintf("Expectativa de %d sockets conectados em %s\n", qtd, interval.String()))

	var i int64
	for i = 1; i <= qtd; i = i + 1 {
		// Montando JWT
		jwtStr := createJWT(initialID + i - 1)

		go connectWebSocket(initialID+i, uri+"/"+jwtStr)

		// Exibindo o resultados parciais
		if (i * 100 / qtd) > nextSample {
			fmt.Printf("%d conexões - %d%s - %s\n", i, nextSample, "%", time.Now().Sub(ini).String())
			nextSample = nextSample + ccSample
			// recalcular throttle
			if qtd > 1 {
				wait = recalcThrottle(qtd-i, ini, interval)
			}
		}
		if wait > 0 {
			time.Sleep(wait)
		}
	}

	fmt.Printf("%d conexões - %s \n", i-1, "100%\n")

	fim := time.Now()
	intervalo := getIntervalo(ini, fim)
	log.Println(" - WSCONNECT - Finalizado em ", intervalo)
	log.Println()
	log.Println("Os clientes estão conectados, tecle <ENTER> para desconectar todos eles")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

}

func connectWebSocket(codCli int64, uri string) {
	ws, err := websocket.Dial(uri, "", "http://localhost")
	if err != nil {
		log.Fatal("Falha ao conectar com socket: ", err)
	}

	var msg []byte
	_, err = ws.Read(msg)
	fmt.Println(fmt.Sprintf("Desconexão, ID=%d, Err=%s", codCli, err.Error()))
}

func createJWT(i int64) string {
	var (
		t   *jwt.Token
		s   string
		err error
	)

	id := fmt.Sprint(i)

	t = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      id,
		"entity":  "driver",
		"App-Key": "appkey_driver_" + id,
	})

	if s, err = t.SignedString([]byte(ccSecret)); err != nil {
		log.Fatalln("Erro criar o JWT:", err)
	}

	return s
}

func recalcThrottle(qtdRestante int64, ini time.Time, intervalo time.Duration) time.Duration {
	agora := time.Now()
	if tempoGasto := agora.Sub(ini); tempoGasto > intervalo {
		// intervalo ultrapassado, vamos retirar o sleep para acelerar ao máximo o processamento.
		return 0
	}

	fim := ini.Add(intervalo)
	return time.Duration(fim.Sub(agora).Abs().Microseconds()/int64(qtdRestante)) * time.Microsecond
}

func getIntervalo(ini time.Time, fim time.Time) string {
	diferenca := fim.Sub(ini)

	horas := int(diferenca.Hours())
	minutos := int(diferenca.Minutes()) % 60
	segundos := int(diferenca.Seconds()) % 60

	return fmt.Sprintf("%dh %dm %ds", horas, minutos, segundos)
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"time"

	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	log "github.com/sirupsen/logrus"
)

type conf struct {
	IP                string
	TimeToAproveClaim time.Duration
	TimeToVerify      time.Duration
}

type counter struct {
	sync.Mutex
	n int
}

type pendingClaimsMap struct {
	sync.Mutex
	Map map[int]time.Time
}

var c = conf{}
var claimCounter = counter{n: 0}

func main() {
	// init
	parseFlags()
	if c.IP == "error" {
		panic("IP flag is mandatory")
	}
	claimRes, credRes, pubDataRes := getMockup()
	pendingClaims := &pendingClaimsMap{Map: make(map[int]time.Time)}
	// ISSUER ENDPOINTS

	// /claim/request
	http.HandleFunc("/claim/request", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Endpoint /claim/request reached")
		claimCounter.Lock()
		tracker := claimCounter.n
		claimCounter.n++
		claimCounter.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(issuerMsg.ResClaimRequest{
			Id: tracker,
		}); err != nil {
			log.Error("Error sending /claim/request response: " + err.Error())
		}
		pendingClaims.Lock()
		pendingClaims.Map[tracker] = time.Now()
		pendingClaims.Unlock()
	})

	// /claim/status/
	http.HandleFunc("/claim/status/", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Endpoint /claim/status/:id reached")
		tra := strings.TrimPrefix(r.URL.Path, "/claim/status/")
		tracker, err := strconv.Atoi(tra)
		if err != nil {
			w.WriteHeader(400)
			log.Error("/claim/status/:id id is not int")
			return
		}
		pendingClaims.Lock()
		t, ok := pendingClaims.Map[tracker]
		pendingClaims.Unlock()
		if !ok {
			w.WriteHeader(404)
			log.Error("/claim/status/:id NOT FOUND")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if time.Since(t) <= c.TimeToAproveClaim {
			if err := json.NewEncoder(w).Encode(issuerMsg.ResClaimStatus{
				Status: issuerMsg.RequestStatusPending,
			}); err != nil {
				log.Error("Error sending /claim/status/:id response: " + err.Error())
				return
			}
			log.Info("claim status: PENDING")
		} else {
			if _, err := w.Write([]byte(claimRes)); err != nil {
				log.Error("Error sending /claim/status/:id response: " + err.Error())
			} else {
				pendingClaims.Lock()
				delete(pendingClaims.Map, tracker)
				pendingClaims.Unlock()
				log.Info("claim status: SENDED")
			}
		}
	})

	// /claim/credential
	http.HandleFunc("/claim/credential", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Endpoint /claim/credential reached")
		var body issuerMsg.ReqClaimCredential
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(400)
			log.Error("Error parsing body for /claim/credential: " + err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		log.Info("claim credential: READY")
		if _, err := w.Write([]byte(credRes)); err != nil {
			log.Error("Error sending /claim/credential response: " + err.Error())
		}
	})

	// /idenpublicdata
	http.HandleFunc("/idenpublicdata/118NZoexLLTgiApGVod8cGXRTeae1a9RqvYaJM5cq4/state/0xdc32e1028f17499f15f37fc4bc1070aa9b28d827b739af456733fa99c92b7827", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Endpoint /idenpublicdata reached")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(pubDataRes)); err != nil {
			log.Error("Error sending /claim/request response: " + err.Error())
		}
	})

	// VERIFIER ENDPOINTS
	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Endpoint /verify")
		w.WriteHeader(200)
	})

	fmt.Println("server running at", c.IP+":1234")
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

func parseFlags() {
	var ttac int
	var ttv int
	flag.IntVar(&ttac, "aprovetime", 60, "Time that takes to aprove a claim request(in seconds)")
	flag.IntVar(&ttv, "verifytime", 60, "Time that takes to verify a claim (in seconds)")
	flag.StringVar(&c.IP, "ip", "error", "IP of the machine where this software will run")

	flag.Parse()
	c.TimeToAproveClaim = time.Duration(ttac) * time.Second
	c.TimeToVerify = time.Duration(ttv) * time.Second
}

func getMockup() (string, string, string) {
	pubDataUrl := "http://" + c.IP + ":1234/idenpublicdata/"
	const claimRes = `{
		"status": "approved",
		"claim": "0x0000000000000000000000003131347674793255664d6b74714e50645a687300567151673948594b67546e7a507058644537796b655146000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	}`
	credRes := `{
		"status": "ready",
		"credential": {
			"Id": "118NZoexLLTgiApGVod8cGXRTeae1a9RqvYaJM5cq4",
			"IdenStateData": {
				"BlockTs": 1582637420,
				"BlockN": 2240421,
				"IdenState": "0x6e7c6798c63a9f168ac705ce6cafd1f0076798cb258bac1d3509c0b8c3c7ad01"
			},
			"MtpClaim": "0x00050000000000000000000000000000000000000000000000000000000000179242e667845d558d6d9061b7d406f9b2fecb9dd98ad0f205b4576b8b2336a92cd9b65acbed05b78bb199ec5de3225281e38ccbd8add32c65f715603862c68c0476c47edbbc50e3c510182b7b4ed49f5968bf644978cc4d8a2b0218f7ff1f461b5c4fa26c1eac035b2eea47a401f953e62d2bbad53800a1e390937b083b168124",
			"Claim": "0x0000000000000000000000003131347674793255664d6b74714e50645a687300567151673948594b67546e7a507058644537796b655146000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			"RevocationsTreeRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"RootsTreeRoot": "0x8641ba923381e34f94be83b444a5cb7bff0a470d92a2d7cf99f5ca0f98405b21",
			"IdenPubUrl": "` + pubDataUrl + `"
		}
	}`
	const pubDataRes = `{
		"IdenState": "0x6e7c6798c63a9f168ac705ce6cafd1f0076798cb258bac1d3509c0b8c3c7ad01",
		"ClaimsTreeRoot": "0x3b3b1d855dbbf4861c5e94c4d108bf00657cf6fca7f3cb429e50f44fdc33ce12",
		"RevocationsTreeRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"RevocationsTree": "CyAAY3VycmVudHJvb3QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"RootsTreeRoot": "0x8641ba923381e34f94be83b444a5cb7bff0a470d92a2d7cf99f5ca0f98405b21",
		"RootsTree": "IAEBhkG6kjOB40+UvoO0RKXLe/8KRw2SotfPmfXKD5hAWyEBRkCv2e9mhBMMXINP3tpGgjG4x2vuHff7myENW8iwcRQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAsgAGN1cnJlbnRyb290hkG6kjOB40+UvoO0RKXLe/8KRw2SotfPmfXKD5hAWyE="
	}`
	return claimRes, credRes, pubDataRes
}

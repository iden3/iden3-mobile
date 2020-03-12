package mockupserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"time"

	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	log "github.com/sirupsen/logrus"
)

type Conf struct {
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

func Serve(c *Conf) error {
	// init
	claimCounter := counter{n: 1}
	claimRes, credRes, pubDataRes := getMockup(c)
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
	http.HandleFunc("/idenpublicdata/114HNY4C7NrKMQ3XZ7GPLdaQqAQ2TjxgFtLEq312nf/state/0xaaada0c31752c0e794b64cd65260d8d7506e3fe19ed7e0341cc925d1bb85530e", func(w http.ResponseWriter, r *http.Request) {
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
		return err
	}
	return nil
}

func getMockup(c *Conf) (string, string, string) {
	pubDataUrl := "http://" + c.IP + ":1234/idenpublicdata/"
	const claimRes = `{"status":"approved","claim":"0x00000000000000000000000035623166343339333433356366623862376331003537383164613230313837633661393966363033373638303364643933393000613864323961613765623031626638366531366462636161663366643365000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}`
	credRes := `{"status":"ready","credential":{"Id":"114HNY4C7NrKMQ3XZ7GPLdaQqAQ2TjxgFtLEq312nf","IdenStateData":{"BlockTs":1583931881,"BlockN":2326694,"IdenState":"0xaaada0c31752c0e794b64cd65260d8d7506e3fe19ed7e0341cc925d1bb85530e"},"MtpClaim":"0x0001000000000000000000000000000000000000000000000000000000000001693a465c114dfe2f02256694c51c5b965b99368d8530f060f020ef5cbad8d81f","Claim":"0x00000000000000000000000035623166343339333433356366623862376331003537383164613230313837633661393966363033373638303364643933393000613864323961613765623031626638366531366462636161663366643365000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","RevocationsTreeRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","RootsTreeRoot":"0x85a7890b63cad6b8ab3b0e3f81de58ccf13a71a716a9c4bb119ea9c341031529","IdenPubUrl":"` + pubDataUrl + `"}}`
	const pubDataRes = `{"IdenState":"0x0c58c40ba4a6152fb439d71d432dc9c07257ad0d7408f033c1ac34d9c2183e11","ClaimsTreeRoot":"0x9555a73ba82547fccb5d46d6e74f0d2961fd75f139e237371c8aa46ac9463323","RevocationsTreeRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","RevocationsTree":"CyAAY3VycmVudHJvb3QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","RootsTreeRoot":"0x1d6f7477a7224e12f140b931974717665883d702c21b711499479e05fdec2424","RootsTree":"IAEBHW90d6ciThLxQLkxl0cXZliD1wLCG3EUmUeeBf3sJCQB2bDCDimA50VqkY+oWiYq7jxWJGQwYVtR90BdtdLyAwwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAsgAGN1cnJlbnRyb290HW90d6ciThLxQLkxl0cXZliD1wLCG3EUmUeeBf3sJCQ="}`
	return claimRes, credRes, pubDataRes
}

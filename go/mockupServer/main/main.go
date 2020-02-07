package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"time"

	"github.com/google/uuid"
)

type (
	verifierResponse struct {
		Done    bool
		Success bool
	}

	issuerResponse struct {
		Done    bool
		Success bool
		Claim   string
	}

	conf struct {
		IP               string
		TimeToBuildClaim time.Duration
		TimeToVerify     time.Duration
	}
)

var c = conf{}

func main() {
	parseFlags()
	if c.IP == "error" {
		panic("IP flag is mandatory")
	}
	pendingClaims := make(map[string]time.Time)
	pendingVerifications := make(map[string]time.Time)
	http.HandleFunc("/issueClaim", func(w http.ResponseWriter, r *http.Request) {
		tracker := uuid.New().String()
		if _, err := w.Write([]byte("http://" + c.IP + ":1234/getClaim?tracker=" + tracker)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("/issueClaim: ERROR SENDING TRACKER")
		} else {
			fmt.Println("/issueClaim: received claim request:", tracker)
			pendingClaims[tracker] = time.Now()
		}
	})

	http.HandleFunc("/getClaim", func(w http.ResponseWriter, r *http.Request) {
		tracker := r.URL.Query().Get("tracker")
		if value, ok := pendingClaims[tracker]; ok && time.Since(value) > c.TimeToBuildClaim {
			j, err := json.Marshal(issuerResponse{
				Done:    true,
				Success: true,
				Claim:   uuid.New().String(),
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getClaim: ERROR BUILDING CLAIM RESPONSE")
			}
			if _, err := w.Write(j); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getClaim: ERROR SENDING CLAIM")
			} else {
				fmt.Println("/getClaim: claim issued:", tracker)
				pendingClaims[tracker] = time.Now()
			}
		} else {
			j, err := json.Marshal(issuerResponse{
				Done:    false,
				Success: false,
				Claim:   "",
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getClaim: ERROR BUILDING CLAIM RESPONSE")
			}
			if _, err := w.Write(j); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getClaim: ERROR SENDING CLAIM")
			}
			fmt.Println("/getClaim: CLAIM NOT BUIDL YET!")
		}
	})

	http.HandleFunc("/verifyClaim", func(w http.ResponseWriter, r *http.Request) {
		tracker := uuid.New().String()
		if _, err := w.Write([]byte("http://" + c.IP + ":1234/getVerification?tracker=" + tracker)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("/verifyClaim: ERROR SENDING TRACKER")
		} else {
			fmt.Println("/verifyClaim: received verification request:", tracker)
			pendingVerifications[tracker] = time.Now()
		}
	})

	http.HandleFunc("/getVerification", func(w http.ResponseWriter, r *http.Request) {
		tracker := r.URL.Query().Get("tracker")
		if value, ok := pendingVerifications[tracker]; ok && time.Since(value) > c.TimeToVerify {
			j, err := json.Marshal(verifierResponse{
				Done:    true,
				Success: true,
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getVerification: ERROR BUILDING VERIFICATION RESPONSE")
			}
			if _, err := w.Write(j); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getVerification: ERROR SENDING VERIFICATION")
			} else {
				fmt.Println("/getVerification: claim verified:", tracker)
				pendingVerifications[tracker] = time.Now()
			}
		} else {
			j, err := json.Marshal(verifierResponse{
				Done:    false,
				Success: false,
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getVerification: ERROR BUILDING VERIFICATION RESPONSE")
			}
			if _, err := w.Write(j); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("/getVerification: ERROR SENDING VERIFICATION")
			}
			fmt.Println("/getVerification: NOT VERIFIED YET!")
		}
	})

	fmt.Println("server running at", c.IP+":1234")
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

func parseFlags() {
	var ttbc int
	var ttv int
	flag.IntVar(&ttbc, "issuetime", 60, "Time that takes to build a claim (in seconds)")
	flag.IntVar(&ttv, "verifytime", 60, "Time that takes to verify a claim (in seconds)")
	flag.StringVar(&c.IP, "ip", "error", "IP of the machine where this software will run")

	flag.Parse()
	c.TimeToBuildClaim = time.Duration(ttbc) * time.Second
	c.TimeToVerify = time.Duration(ttv) * time.Second
}

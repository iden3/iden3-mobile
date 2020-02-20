package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"time"

	"github.com/google/uuid"
	"github.com/iden3/go-iden3-core/core"
	"github.com/iden3/go-iden3-core/core/claims"
	"github.com/iden3/go-iden3-core/core/proof"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity/issuer"
	"github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-core/merkletree"
)

type (
	verifierResponse struct {
		Done    bool
		Success bool
	}

	issuerResponse struct {
		Done       bool
		Success    bool
		Credential *proof.CredentialExistence
	}

	conf struct {
		IP               string
		TimeToBuildClaim time.Duration
		TimeToVerify     time.Duration
	}
)

var c = conf{}

func main() {
	// init
	parseFlags()
	if c.IP == "error" {
		panic("IP flag is mandatory")
	}
	is := initIssuer()
	pendingClaims := make(map[string]time.Time)
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
		cred := genRandomCredential(is)
		if value, ok := pendingClaims[tracker]; ok && time.Since(value) > c.TimeToBuildClaim {
			j, err := json.Marshal(issuerResponse{
				Done:       true,
				Success:    true,
				Credential: cred,
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
		j, err := json.Marshal(verifierResponse{
			Done:    true,
			Success: true,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("/verifyClaim: ERROR BUILDING VERIFICATION RESPONSE")
		}
		if _, err := w.Write(j); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("/verifyClaim: ERROR SENDING VERIFICATION")
		} else {
			fmt.Println("/verifyClaim: claim verified:")
		}
	})

	fmt.Println("server running at", c.IP+":1234")
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

func initIssuer() *issuer.Issuer {
	cfg := issuer.ConfigDefault
	storage := db.NewMemoryStorage()
	ksStorage := keystore.MemStorage([]byte{})
	keyStore, err := keystore.NewKeyStore(&ksStorage, keystore.LightKeyStoreParams)
	if err != nil {
		panic(err)
	}
	kOp, err := keyStore.NewKey([]byte("pass"))
	if err != nil {
		panic(err)
	}
	err = keyStore.UnlockKey(kOp, []byte("pass"))
	if err != nil {
		panic(err)
	}
	is, err := issuer.New(cfg, kOp, []merkletree.Entrier{}, storage, keyStore, nil, nil)
	if err != nil {
		panic(err)
	}
	return is
}

func genRandomCredential(is *issuer.Issuer) *proof.CredentialExistence {
	var indexSlot [claims.IndexSlotBytes]byte
	var dataSlot [claims.DataSlotBytes]byte
	indexSlotHex, err := hex.DecodeString("292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2a292a2a2a2a2a2a2a2a2b")
	if err != nil {
		panic(err)
	}
	dataSlotHex, err := hex.DecodeString("564040404040404040405640404040404040404056404040404040404040564040404040404040405640404040404040404056404040404040404040564040404040404040405640404040404040404056404040404040404040564040404040404040405640404040404040404056404040404040404059")
	if err != nil {
		panic(err)
	}
	copy(indexSlot[:], indexSlotHex[:claims.IndexSlotBytes])
	copy(dataSlot[:], dataSlotHex[:claims.DataSlotBytes])
	claim := claims.NewClaimBasic(indexSlot, dataSlot, 5678)
	id, err := core.IDFromString("11AVZrKNJVqDJoyKrdyaAgEynyBEjksV5z2NjZoPxf")
	if err != nil {
		panic(err)
	}
	return &proof.CredentialExistence{
		Id:                  &id,
		IdenStateData:       proof.IdenStateData{},
		MtpClaim:            &merkletree.Proof{},
		Claim:               claim.Entry(),
		RevocationsTreeRoot: &merkletree.Hash{},
		RootsTreeRoot:       &merkletree.Hash{},
		IdenPubUrl:          "http://TODO",
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

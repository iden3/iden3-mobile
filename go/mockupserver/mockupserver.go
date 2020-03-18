package mockupserver

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/iden3/go-iden3-core/components/idenpuboffchain"
	idenpuboffchainwriterhttp "github.com/iden3/go-iden3-core/components/idenpuboffchain/writerhttp"
	"github.com/iden3/go-iden3-core/components/idenpubonchain"
	"github.com/iden3/go-iden3-core/components/verifier"
	"github.com/iden3/go-iden3-core/core/claims"
	"github.com/iden3/go-iden3-core/db"
	"github.com/iden3/go-iden3-core/identity/issuer"
	"github.com/iden3/go-iden3-core/keystore"
	"github.com/iden3/go-iden3-core/merkletree"
	"github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	issuerMsg "github.com/iden3/go-iden3-servers-demo/servers/issuerdemo/messages"
	verifierMsg "github.com/iden3/go-iden3-servers-demo/servers/verifier/messages"
	"github.com/iden3/go-iden3-servers/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

func ShouldBindJSONValidate(c *gin.Context, v interface{}) error {
	if err := c.ShouldBindJSON(&v); err != nil {
		handlers.Fail(c, "cannot parse json body", err)
		return err
	}
	if err := validator.New().Struct(v); err != nil {
		handlers.Fail(c, "cannot validate json body", err)
		return err
	}
	return nil
}

type Requests struct {
	rw       sync.RWMutex
	n        int
	pending  map[int]issuerMsg.Request
	approved map[int]issuerMsg.Request
	// rejected map[int]issuerMsg.Request
}

func NewRequests() *Requests {
	return &Requests{
		n:        0,
		pending:  make(map[int]issuerMsg.Request),
		approved: make(map[int]issuerMsg.Request),
	}
}

func (r *Requests) Approve(id int, claim merkletree.Entrier) error {
	r.rw.Lock()
	defer r.rw.Unlock()
	request, ok := r.pending[id]
	if !ok {
		return fmt.Errorf("Request id: %v not found", id)
	}
	delete(r.pending, id)
	request.Claim = claim.Entry()
	request.Status = issuerMsg.RequestStatusApproved
	r.approved[id] = request
	return nil
}

func (r *Requests) Add(value string) int {
	r.rw.Lock()
	defer r.rw.Unlock()
	r.n += 1
	request := messages.Request{
		Id:     r.n,
		Value:  value,
		Status: messages.RequestStatusPending,
	}
	r.pending[request.Id] = request
	return r.n
}

func (r *Requests) Get(id int) (*messages.Request, error) {
	r.rw.RLock()
	defer r.rw.RUnlock()
	if request, ok := r.pending[id]; ok {
		return &request, nil
	}
	if request, ok := r.approved[id]; ok {
		return &request, nil
	}
	return nil, fmt.Errorf("Request id: %v not found", id)
}

type Conf struct {
	IP                string
	Port              string
	TimeToAproveClaim time.Duration
	TimeToPublish     time.Duration
}

func NewIssuer(t *testing.T, idenPubOnChain idenpubonchain.IdenPubOnChainer,
	idenPubOffChainWrite idenpuboffchain.IdenPubOffChainWriter) *issuer.Issuer {
	cfg := issuer.ConfigDefault
	storage := db.NewMemoryStorage()
	ksStorage := keystore.MemStorage([]byte{})
	keyStore, err := keystore.NewKeyStore(&ksStorage, keystore.LightKeyStoreParams)
	require.Nil(t, err)
	kOp, err := keyStore.NewKey([]byte("pass"))
	require.Nil(t, err)
	err = keyStore.UnlockKey(kOp, []byte("pass"))
	require.Nil(t, err)
	_, err = issuer.Create(cfg, kOp, []claims.Claimer{}, storage, keyStore)
	require.Nil(t, err)
	is, err := issuer.Load(storage, keyStore, idenPubOnChain, idenPubOffChainWrite)
	require.Nil(t, err)
	return is
}

func Serve(t *testing.T, cfg *Conf, idenPubOnChain idenpubonchain.IdenPubOnChainer) *http.Server {
	idenPubOffChainWrite, err := idenpuboffchainwriterhttp.NewIdenPubOffChainWriteHttp(
		idenpuboffchainwriterhttp.NewConfigDefault(fmt.Sprintf("http://%v:%v/idenpublicdata/", cfg.IP, cfg.Port)),
		db.NewMemoryStorage(),
	)
	require.Nil(t, err)
	is := NewIssuer(t, idenPubOnChain, idenPubOffChainWrite)
	requests := NewRequests()
	verif := verifier.New(idenPubOnChain)

	// Publish and sync issuer state every 2 seconds
	go func() {
		for {
			err := is.PublishState()
			if err != nil {
				log.WithError(err).Warn("Issuer.PublishState()")
			}
			err = is.SyncIdenStatePublic()
			if err != nil {
				log.WithError(err).Error("Issuer.SyncIdenStatePublic()")
			}
			time.Sleep(cfg.TimeToPublish)
		}
	}()

	api := gin.Default()
	api.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": "404 page not found",
		})
	})

	// ISSUER ENDPOINTS

	api.POST("/claim/request", func(c *gin.Context) {
		var req issuerMsg.ReqClaimRequest
		if err := ShouldBindJSONValidate(c, &req); err != nil {
			return
		}
		id := requests.Add(req.Value)
		// Approve request and issue claim after c.TimeToAproveClaim duration
		go func() {
			time.Sleep(cfg.TimeToAproveClaim)
			indexSlot, valueSlot := [claims.IndexSlotLen]byte{}, [claims.ValueSlotLen]byte{}
			copy(indexSlot[:], []byte(req.Value))
			claim := claims.NewClaimBasic(indexSlot, valueSlot)

			// Issue Claim
			if err := is.IssueClaim(claim); err != nil {
				log.WithError(err).WithField("value", req.Value).Error("SRV Issuer.IssueClaim()")
				return

			}
			err := requests.Approve(id, claim)
			if err != nil {
				log.WithError(err).WithField("value", req.Value).Info("SRV requests.Approve()")
			}
		}()
		c.JSON(200, issuerMsg.ResClaimRequest{
			Id: id,
		})
	})

	api.GET("/claim/status/:id", func(c *gin.Context) {
		var uri struct {
			Id int `uri:"id"`
		}
		if err := c.ShouldBindUri(&uri); err != nil {
			handlers.Fail(c, "cannot validate uri", err)
			return
		}
		request, err := requests.Get(uri.Id)
		if err != nil {
			handlers.Fail(c, "Requests.Get()", err)
			return
		}
		c.JSON(200, issuerMsg.ResClaimStatus{
			Status: request.Status,
			Claim:  request.Claim,
		})
	})

	api.POST("/claim/credential", func(c *gin.Context) {
		var req issuerMsg.ReqClaimCredential
		if err := ShouldBindJSONValidate(c, &req); err != nil {
			return
		}
		// Generate Credential Existence
		credential, err := is.GenCredentialExistence(claims.NewClaimGeneric(req.Claim))
		status := issuerMsg.ClaimtStatusReady
		if err == issuer.ErrClaimNotYetInOnChainState {
			log.Debug("Issuer.GenCredentialExistence -> ErrClaimNotYetInOnChainState")
			status = issuerMsg.ClaimtStatusNotYet
			credential = nil
		} else if err == issuer.ErrIdenStateOnChainZero {
			log.Debug("Issuer.GenCredentialExistence -> ErrIdenStateOnChainZero")
			status = issuerMsg.ClaimtStatusNotYet
			credential = nil
		} else if err != nil {
			handlers.Fail(c, "Issuer.GenCredentialExistence()", err)
			return
		}
		c.JSON(200, issuerMsg.ResClaimCredential{
			Status:     status,
			Credential: credential,
		})
	})

	_handleGetIdenPublicData := func(c *gin.Context, state *merkletree.Hash) {
		data, err := idenPubOffChainWrite.GetPublicData(state)
		if err != nil {
			handlers.Fail(c, "idenPubOffChainWrite.GetPublicData()", err)
			return
		}
		c.JSON(200, data)
	}

	api.GET(fmt.Sprintf("/idenpublicdata/%s", is.ID()), func(c *gin.Context) {
		_handleGetIdenPublicData(c, nil)
	})

	api.GET(fmt.Sprintf("/idenpublicdata/%s/state/:state", is.ID()), func(c *gin.Context) {
		var uri struct {
			State string `uri:"state"`
		}
		if err := c.ShouldBindUri(&uri); err != nil {
			handlers.Fail(c, "cannot validate uri", err)
			return
		}
		var state merkletree.Hash
		if err := state.UnmarshalText([]byte(uri.State)); err != nil {
			handlers.Fail(c, "cannot unmarshal state", err)
			return
		}
		_handleGetIdenPublicData(c, &state)
	})

	// VERIFIER ENDPOINTS

	api.POST("/verify", func(c *gin.Context) {
		var req verifierMsg.ReqVerify
		if err := c.ShouldBindJSON(&req); err != nil {
			handlers.Fail(c, "cannot parse json body", err)
			return
		}
		err := verif.VerifyCredentialValidity(req.CredentialValidity, 30*time.Minute)
		if err != nil {
			handlers.Fail(c, "VerifyCredentialValidity()", err)
			return
		}

		c.JSON(200, gin.H{})
	})

	server := &http.Server{Addr: fmt.Sprintf("%v:%v", cfg.IP, cfg.Port), Handler: api}

	go func() {
		if err := ListenAndServe(server, "Service"); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	return server
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err := tc.SetKeepAlive(true); err != nil {
		return nil, err
	}
	if err := tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		return nil, err
	}
	return tc, nil
}

func ListenAndServe(srv *http.Server, name string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Infof("%s API is ready at %v", name, addr)
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

package sim

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
)

const (
	defaultTokenExpiry = time.Hour
	tokenSigningAlg    = jose.HS256
)

type tokenClaims struct {
	NodeID int64  `json:"nodeId"`
	Type   string `json:"type"`
}

type tokenIssuer struct {
	signingKey []byte
	expiry     time.Duration
}

func newTokenIssuer() (*tokenIssuer, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}
	return &tokenIssuer{
		signingKey: key,
		expiry:     defaultTokenExpiry,
	}, nil
}

func (ti *tokenIssuer) issue(nodeID int64) (token string, expiresIn int, err error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: tokenSigningAlg, Key: ti.signingKey}, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create signer: %w", err)
	}

	now := time.Now()
	claims := jwt.Claims{
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(ti.expiry)),
	}
	custom := tokenClaims{
		NodeID: nodeID,
		Type:   "bos_node",
	}

	raw, err := jwt.Signed(signer).Claims(claims).Claims(custom).Serialize()
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}
	return raw, int(ti.expiry.Seconds()), nil
}

func (ti *tokenIssuer) validate(tokenStr string) (tokenClaims, error) {
	tok, err := jwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{tokenSigningAlg})
	if err != nil {
		return tokenClaims{}, fmt.Errorf("parse token: %w", err)
	}

	var stdClaims jwt.Claims
	var custom tokenClaims
	if err := tok.Claims(ti.signingKey, &stdClaims, &custom); err != nil {
		return tokenClaims{}, fmt.Errorf("verify token: %w", err)
	}

	if err := stdClaims.ValidateWithLeeway(jwt.Expected{}, 0); err != nil {
		return tokenClaims{}, fmt.Errorf("token expired or invalid: %w", err)
	}

	if custom.Type != "bos_node" {
		return tokenClaims{}, errors.New("invalid token type")
	}

	return custom, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// handleToken implements the OAuth2 Client Credentials Grant (RFC 6749 §4.4).
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	logger := s.loggerFor(r)

	if err := r.ParseForm(); err != nil {
		writeError(w, errInvalidRequest)
		logger.Debug("failed to parse form", zap.Error(err))
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "client_credentials" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":             "unsupported_grant_type",
			"error_description": "only client_credentials grant type is supported",
		})
		return
	}

	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if clientID == "" || clientSecret == "" {
		writeError(w, errUnauthorized)
		logger.Debug("missing client credentials")
		return
	}

	secretBytes, err := base64.StdEncoding.DecodeString(clientSecret)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("invalid base64 in client_secret", zap.Error(err))
		return
	}
	secretHash := sha256.Sum256(secretBytes)

	parsedClientID, err := strconv.ParseInt(clientID, 10, 64)
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("invalid client_id", zap.Error(err))
		return
	}

	var nodeID int64
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		node, err := tx.GetNodeBySecretHash(r.Context(), secretHash[:])
		if err != nil {
			return err
		}
		if node.ID != parsedClientID {
			return errUnauthorized
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		writeError(w, errUnauthorized)
		logger.Debug("token auth failed", zap.Error(err))
		return
	}

	token, expiresIn, err := s.tokenIssuer.issue(nodeID)
	if err != nil {
		writeError(w, errInternal)
		logger.Error("failed to issue token", zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	})
}

func parseBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", errors.New("invalid authorization scheme")
	}
	token := auth[len(prefix):]
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

func (s *Server) authenticateNode(r *http.Request) (nodeID int64, err error) {
	tokenStr, err := parseBearerToken(r)
	if err != nil {
		return 0, err
	}
	claims, err := s.tokenIssuer.validate(tokenStr)
	if err != nil {
		return 0, err
	}
	return claims.NodeID, nil
}


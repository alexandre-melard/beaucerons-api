package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/alexandre-melard/beaucerons/api/utils"
	"github.com/form3tech-oss/jwt-go"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

func CheckPermission(w http.ResponseWriter, r *http.Request, scope string) error {
	authHeaderParts := strings.Split(r.Header.Get("Authorization"), " ")
	token := authHeaderParts[1]
	if !checkScope(scope, token) {
		utils.ResponseJSON("Forbidden", w, http.StatusForbidden)
		return fmt.Errorf("forbiden, scope %s is not present in claims", scope)
	}
	return nil
}

func CheckKey(token *jwt.Token) (interface{}, error) {
	// Verify 'aud' claim
	aud := os.Getenv("AUTH0_AUDIENCE")
	checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
	if !checkAud {
		return token, fmt.Errorf("invalid audience")
	}
	// Verify 'iss' claim
	iss := "https://" + os.Getenv("AUTH0_DOMAIN") + "/"
	checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
	if !checkIss {
		return token, fmt.Errorf("invalid issuer")
	}

	cert, err := getPemCert(token)
	if err != nil {
		panic(err.Error())
	}

	result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	return result, nil
}

func checkScope(scopeToCheck string, tokenString string) bool {
	claims := jwt.MapClaims{}
	token, _ := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		cert, err := getPemCert(token)
		if err != nil {
			return nil, err
		}
		result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		return result, nil
	})
	if token.Valid {
		scope := fmt.Sprintf("%v", claims["scope"])
		result := strings.Split(scope, " ")
		for i := range result {
			if result[i] == scopeToCheck {
				return true
			}
		}
	}
	return false
}

func getPemCert(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get("https://" + os.Getenv("AUTH0_DOMAIN") + "/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k, _ := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := fmt.Errorf("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}

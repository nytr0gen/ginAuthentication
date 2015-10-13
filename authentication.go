package ginAuthentication

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"crypto/aes"
	"encoding/base64"
	"io"
	"crypto/rand"
	"crypto/cipher"
	"bytes"
	"time"
)

type checkCredentials func(authenticationCredentials) (valid bool, err error)

type AuthenticationEngine struct {
	aesKey []byte
	cookieName string
	fn checkCredentials
	cookieExpirationTime time.Time
}
type authenticationCredentials struct{
	username string
	password string
}

func New(params AuthenticationEngine) (engine *AuthenticationEngine,err error)  {
	if len(params.aesKey) != 32 {
		return nil,errors.New("aesKey must be 32bytes")
	}
	return &AuthenticationEngine{cookieName:params.cookieName,fn:params.fn},nil
}

func (engine *AuthenticationEngine) Validate(credentials authenticationCredentials) (bool,error){
	valid,err:=engine.fn(credentials)
	return valid,err
}

func (engine *AuthenticationEngine) ValidateAndSetCookie(credentials authenticationCredentials,c *gin.Context) (bool,error){
	valid,err:= engine.fn(credentials)
	if err!=nil{
		return false,err
	}
	encryptedCookie,err:=encryptAES(engine.aesKey,[]byte("loggedIn=true"))
	if(err){
		return false,err
	}
	cookie := http.Cookie{Name: engine.cookieName, Value:encryptedCookie, Expires: engine.cookieExpirationTime}
	http.SetCookie(cookie, &cookie)
	return valid,nil
}

func (engine *AuthenticationEngine) ValidationMiddleware (notAuthenticatedRoute string) {
	return func(c *gin.Context) {
		cookieString,err:=c.Request.Cookie(engine.cookieName)
		if err!=nil{
			c.Redirect(http.StatusForbidden,notAuthenticatedRoute)
		}
		value,err:=decryptAES(engine.aesKey,[]bytes(cookieString))
	    if err!=nil || !bytes.Equal(value,[]byte("loggedIn=true")){
			c.Redirect(http.StatusForbidden,notAuthenticatedRoute)
		}else{
			c.Next()
		}
	}
}

func encryptAES(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decryptAES(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}


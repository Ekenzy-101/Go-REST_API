package models

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ekenzy-101/Go-Gin-REST-API/config"
	"github.com/Ekenzy-101/Go-Gin-REST-API/services"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/argon2"
)

type PasswordConfig struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

type User struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"  json:"_id,omitempty"`
	AccountVerified bool               `bson:"accountVerified" json:"accountVerified"`
	Bio             string             `bson:"bio" json:"bio"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt,omitempty"`
	Email           string             `bson:"email" json:"email,omitempty" binding:"email,max=255"`
	FollowerCount   int                `bson:"followerCount" json:"followerCount"`
	FollowingCount  int                `bson:"followingCount" json:"followingCount"`
	Gender          string             `bson:"gender" json:"gender"`
	Image           string             `bson:"image" json:"image"`
	Name            string             `bson:"name" json:"name,omitempty" binding:"required,alpha,max=50"`
	Password        string             `bson:"password" json:"password,omitempty"  binding:"required,min=6"`
	PostCount       int                `bson:"postCount" json:"postCount"`
	Posts           []Post             `bson:"posts" json:"posts"`
	PhoneNo         string             `bson:"phoneNo" json:"phoneNo,omitempty"`
	Username        string             `bson:"username" json:"username,omitempty" binding:"username"`
	Website         string             `bson:"website" json:"website"`
}

func (user *User) ComparePassword(password string) (bool, error) {
	parts := strings.Split(user.Password, "$")

	if len(parts) < 4 {
		return false, errors.New("invalid string")
	}

	c := &PasswordConfig{}
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &c.memory, &c.time, &c.threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	c.keyLen = uint32(len(decodedHash))

	comparisonHash := argon2.IDKey([]byte(password), salt, c.time, c.memory, c.threads, c.keyLen)

	return (subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1), nil
}

func (user *User) GenerateToken() (string, error) {
	claims := &services.AccessTokenClaim{
		Email: user.Email,
		ID:    user.ID.Hex(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Second * config.AccessTokenTTLInSeconds).Unix(),
		},
	}

	option := services.JWTOption{
		SigningMethod: jwt.SigningMethodHS256,
		Claims:        claims,
		Secret:        config.AccessTokenSecret,
	}
	return services.SignToken(option)
}

func (user *User) HashPassword() error {
	c := &PasswordConfig{
		time:    1,
		memory:  64 * 1024,
		threads: 4,
		keyLen:  32,
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	hash := argon2.IDKey([]byte(user.Password), salt, c.time, c.memory, c.threads, c.keyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	user.Password = fmt.Sprintf(format, argon2.Version, c.memory, c.time, c.threads, b64Salt, b64Hash)

	return nil
}

func (user *User) NormalizeFields(new bool) {
	user.Email = strings.ToLower(user.Email)
	user.Name = strings.TrimSpace(user.Name)

	if new {
		user.ID = primitive.NewObjectID()
		user.CreatedAt = time.Now()
	}
}

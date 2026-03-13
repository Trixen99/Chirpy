package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPassword_1(t *testing.T) {
	type testCase struct {
		password  string
		password2 string

		expected bool
	}

	t.Run("testing hashing and unhashing", func(t *testing.T) {
		tests := []testCase{
			{"hello", "hello", true},
			{"hello", "bye", false},
		}

		for _, test := range tests {
			hash, err := HashPassword(test.password)
			if err != nil {
				assert.NoError(t, err)
			}
			actual, err := CheckPasswordHash(test.password2, hash)
			if err != nil {
				assert.NoError(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		}

	})

}

func TestJWT_1(t *testing.T) {
	type testCase struct {
		id           uuid.UUID
		key          string
		expiresIn    time.Duration
		stringresult string
	}

	tests := []testCase{
		{uuid.MustParse("35da1215-ff47-4947-9d0a-3a94cb6293ba"), "helloworld", time.Hour * 5, ""},
		{uuid.MustParse("5a5ce86d-86b1-4bff-be02-f78af9433bf1"), "my_super_long_password_test_for_token_generation", time.Hour - (time.Hour*2)*2, ""},
		{uuid.MustParse("c94de4cf-489a-436e-a2fb-ca6d8e538a21"), "these_are_not_the_tokens_you_are_looking_for_(hand_have)", time.Minute * 100000000, ""},
	}

	for i, test := range tests {
		expiredtime := time.Now().Add(test.expiresIn)
		secretstring, err := MakeJWT(test.id, test.key, test.expiresIn)
		if err != nil {
			assert.NoError(t, err)
		}
		tests[i].stringresult = secretstring
		newuuid, err := ValidateJWT(secretstring, test.key)

		if expiredtime.Before(time.Now()) {
			assert.Error(t, err, "token has invalid claims: token is expired")
			continue
		}
		if err != nil {
			assert.NoError(t, err)
		}

		assert.Equal(t, test.id, newuuid)

	}

	for _, test := range tests {
		_, err := ValidateJWT(test.stringresult, "up,up,down,down,left,right,left,right,B,A")
		assert.Error(t, err, "token signature is invalid: signature is invalid")
	}

}

func TestToken_1(t *testing.T) {
	type testCase struct {
		input http.Header

		expected string
	}

	t.Run("testing hashing and unhashing", func(t *testing.T) {
		tests := []testCase{
			{http.Header{"Authorization": []string{"Bearer ${jwtTokenSaul}"}}, "${jwtTokenSaul}"},
			{http.Header{"Authorization": []string{"Bearer ${jwtTokenMike}"}}, "${jwtTokenMike}"},
		}

		for _, test := range tests {
			token, err := GetBearerToken(test.input)
			if err != nil {
				assert.NoError(t, err)
				continue
			}

			assert.Equal(t, test.expected, token)

		}

	})

}

func TestRefreshToken_1(t *testing.T) {
	sofar := make(map[string]string)

	t.Run("testing hashing and unhashing", func(t *testing.T) {

		for i := 0; i < 10; i++ {
			token := MakeRefreshToken()

			assert.NotContains(t, sofar, token)

			sofar[token] = token

		}

	})

}

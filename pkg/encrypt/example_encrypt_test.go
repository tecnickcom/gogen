package encrypt_test

import (
	"fmt"
	"log"

	"github.com/tecnickcom/nurago/pkg/encrypt"
)

// The nonce is random, so the ciphertext differs on every run; these examples
// print the decrypted round-trip result, which is deterministic and verifiable.
func ExampleEncrypt() {
	key := []byte("abcdefghijklmnopqrstuvwxyz012345") // 32 bytes: AES-256

	enc, err := encrypt.Encrypt(key, []byte("secret message"))
	if err != nil {
		log.Fatal(err)
	}

	dec, err := encrypt.Decrypt(key, enc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(dec))

	// Output:
	// secret message
}

// ExampleEncryptWith binds additional authenticated data (AAD) to the payload.
// The same AAD must be supplied when decrypting.
func ExampleEncryptWith() {
	key := []byte("abcdefghijklmnopqrstuvwxyz012345")
	aad := []byte("record-id-42")

	enc, err := encrypt.EncryptWith(key, []byte("secret message"), encrypt.WithAAD(aad))
	if err != nil {
		log.Fatal(err)
	}

	dec, err := encrypt.DecryptWith(key, enc, encrypt.WithAAD(aad))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(dec))

	// Output:
	// secret message
}

func ExampleEncryptSerializeAny() {
	type Payload struct {
		Name  string
		Count int
	}

	key := []byte("abcdefghijklmnopqrstuvwxyz012345")

	enc, err := encrypt.EncryptSerializeAny(key, Payload{Name: "widget", Count: 7})
	if err != nil {
		log.Fatal(err)
	}

	var out Payload

	err = encrypt.DecryptSerializeAny(key, enc, &out)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(out.Name, out.Count)

	// Output:
	// widget 7
}

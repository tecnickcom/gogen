package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
)

func main() {
	pwd := []byte(os.Args[1])
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		log.Fatal(err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(string(hash)), pwd)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(hash))
}

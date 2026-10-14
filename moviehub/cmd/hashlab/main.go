package main

import (
	"fmt"
	"strings"
	"time"

	"moviehub/auth"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	a, err := auth.HashPassword("popcorn2569")
	if err != nil {
		panic(err)
	}

	b, err := auth.HashPassword("popcorn2569")
	if err != nil {
		panic(err)
	}

	fmt.Println("Hash A:", a)
	fmt.Println("Hash B:", b)

	fmt.Println("popcorn2569 ตรงกับ A:", auth.CheckPassword(a, "popcorn2569"))
	fmt.Println("popcorn2569 ตรงกับ B:", auth.CheckPassword(b, "popcorn2569"))
	fmt.Println("popcorn2568 ตรงกับ A:", auth.CheckPassword(a, "popcorn2568"))

	for _, cost := range []int{8, 10, 12} {
		start := time.Now()
		_, err := bcrypt.GenerateFromPassword([]byte("popcorn2569"), cost)
		if err != nil {
			fmt.Println("bcrypt error:", err)
			continue
		}
		fmt.Printf("cost %d ใช้เวลา %v\n", cost, time.Since(start).Round(time.Millisecond))
	}

	_, err = auth.HashPassword(strings.Repeat("a", 73))
	fmt.Println("73 ไบต์:", err)
}

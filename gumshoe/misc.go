package gumshoe

import (
  "log"
  "math/rand"
  "strconv"
)

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func GetInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}

func GetRandom(seed int64) *rand.Rand {
  return rand.New(rand.NewSource(seed))
}

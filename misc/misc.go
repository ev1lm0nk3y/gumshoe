package misc

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
)

var debug = false

func SetDebug() {
  debug = true
}

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

func PrintDebug(s ...interface{}) {
	PrintDebugf("%s", s...)
}

func PrintDebugln(s ...interface{}) {
	PrintDebugf("%s\n", s...)
}

func PrintDebugf(f string, i ...interface{}) {
	if debug {
		log.Printf("[DEBUG] %s", fmt.Sprintf(f, i...))
	}
}

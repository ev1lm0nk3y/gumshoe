package gumshoe

import (
  "log"
)

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func getInt(s string) int {
	r, _ := strconv.Atoi(s)
	return r
}

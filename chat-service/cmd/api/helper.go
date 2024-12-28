package main

import (
	"strconv"

	"github.com/samber/lo"
)

func IntToStringArray(arr []int) []string {
	return lo.Map(arr, func(item int, _ int) string {
		return strconv.Itoa(item)
	})
}

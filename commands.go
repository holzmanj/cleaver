package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

func tokenizeCommand(cmd string) (map[string]string, error) {
	if len(cmd) < 1 {
		return make(map[string]string), errors.New("empty command string")
	}

	tokens := make(map[string]string)

	// first character is always the chain/channel number
	tokens["chain"] = cmd[:1]
	cmd = cmd[1:]

	// iterate through rest of command and break out individual effect changes
	for len(cmd) > 0 {
		chr, _ := utf8.DecodeRuneInString(cmd)
		switch chr {
		case 'c':
			if len(cmd) < 2 {
				return tokens, errors.New("invalid command format")
			}
			tokens["chop"] = cmd[1:2]
			cmd = cmd[2:]

		case 's': // speed change takes two arguments
			if len(cmd) < 3 {
				return tokens, errors.New("invalid command format")
			}
			tokens["speed"] = cmd[1:3]
			cmd = cmd[3:]

		case 'm': // mince takes two arguments
			if len(cmd) < 3 {
				return tokens, errors.New("invalid command format")
			}
			tokens["mince"] = cmd[1:3]
			cmd = cmd[3:]

		case 'p':
			if len(cmd) < 2 {
				return tokens, errors.New("invalid command format")
			}
			tokens["pan"] = cmd[1:2]
			cmd = cmd[2:]

		case 'v':
			if len(cmd) < 2 {
				return tokens, errors.New("invalid command format")
			}
			tokens["volume"] = cmd[1:2]
			cmd = cmd[2:]

		default:
			return tokens, fmt.Errorf("unrecognized command symbol: %c", chr)
		}
	}

	return tokens, nil
}

func charToBase36Int(chr rune) (int, error) {
	if chr >= '0' && chr <= '9' {
		return int(chr) - '0', nil
	}

	if chr >= 'a' && chr <= 'z' {
		return 10 + (int(chr) - 'a'), nil
	}

	if chr >= 'A' && chr <= 'Z' {
		return 10 + (int(chr) - 'A'), nil
	}

	return 0, errors.New("invalid base-36 numeral")
}

func ParseCommand(cmd string) (int, []func(*Chain)) {
	commandEffects := make([]func(*Chain), 0)

	tokMap, err := tokenizeCommand(strings.ToLower(cmd))
	if err != nil {
		fmt.Println(err)
		return 0, commandEffects
	}

	// first get chain index
	var chain int
	if val, ok := tokMap["chain"]; ok {
		chr, _ := utf8.DecodeRuneInString(val)
		chain, err = charToBase36Int(chr)
		if err != nil {
			fmt.Print(err)
			return 0, commandEffects
		}
	}

	// check if command included a chop selection
	if val, ok := tokMap["chop"]; ok {
		chr, _ := utf8.DecodeRuneInString(val)
		chopIdx, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}

		commandEffects = append(commandEffects, func(c *Chain) {
			c.PlayChop(chopIdx)
		})
	}

	// check if command included speed modification
	if val, ok := tokMap["speed"]; ok {
		// get the ratio of the two values provided for new speed
		chr, _ := utf8.DecodeRuneInString(val)
		n, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}
		if n == 0 {
			fmt.Println("cannot play at speed of 0")
			return chain, commandEffects
		}
		val = val[1:]

		chr, _ = utf8.DecodeRuneInString(val)
		d, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}
		if d == 0 {
			fmt.Println("cannot divide by zero")
			return chain, commandEffects
		}

		commandEffects = append(commandEffects, func(c *Chain) {
			c.SetSpeed(n, d)
		})
	}

	// check if command included mince modification
	if val, ok := tokMap["mince"]; ok {
		chr, _ := utf8.DecodeRuneInString(val)
		size, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}
		val = val[1:]

		chr, _ = utf8.DecodeRuneInString(val)
		interval, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}

		commandEffects = append(commandEffects, func(c *Chain) {
			c.Remince(size, interval)
		})
	}

	// check if command included pan modification
	if val, ok := tokMap["pan"]; ok {
		chr, _ := utf8.DecodeRuneInString(val)
		p, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}

		commandEffects = append(commandEffects, func(c *Chain) {
			c.SetPan(p)
		})
	}

	// check if command included volume modification
	if val, ok := tokMap["volume"]; ok {
		chr, _ := utf8.DecodeRuneInString(val)
		v, err := charToBase36Int(chr)
		if err != nil {
			fmt.Println(err)
			return chain, commandEffects
		}

		commandEffects = append(commandEffects, func(c *Chain) {
			c.SetVolume(v)
		})
	}

	return chain, commandEffects
}

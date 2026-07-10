package core

import (
	"errors"
	"net"
	"strconv"
	"time"
)

func EvalAndRespond(cmd *RedisCmd, c net.Conn) error {

	switch cmd.Cmd {

	case "PING":
		return evalPING(cmd.Args, c)

	case "ECHO":
		return evalECHO(cmd.Args, c)

	case "SET":
		return evalSET(cmd.Args, c)

	case "GET":
		return evalGET(cmd.Args, c)

	case "DEL":
		return evalDEL(cmd.Args, c)

	case "EXISTS":
		return evalEXISTS(cmd.Args, c)

	case "EXPIRE":
		return evalEXPIRE(cmd.Args, c)

	default:
		return errors.New("ERR unknown command '" + cmd.Cmd + "'")
	}
}

func evalPING(args []string, c net.Conn) error {

	if len(args) > 1 {
		return errors.New("ERR wrong number of arguments for 'ping' command")
	}

	var resp []byte

	if len(args) == 0 {
		resp = Encode("PONG", true)
	} else {
		resp = Encode(args[0], false)
	}

	_, err := c.Write(resp)
	return err
}

func evalECHO(args []string, c net.Conn) error {

	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'echo' command")
	}

	_, err := c.Write(Encode(args[0], false))
	return err
}

func evalSET(args []string, c net.Conn) error {

	if len(args) != 2 {
		return errors.New("ERR wrong number of arguments for 'set' command")
	}

	setKey(args[0], args[1])

	_, err := c.Write(Encode("OK", false))
	return err
}

func evalGET(args []string, c net.Conn) error {

	if len(args) != 1 {
		return errors.New("ERR wrong number of arguments for 'get' command")
	}

	val, ok := getKey(args[0])
	if !ok {
		_, err := c.Write(Encode(nil, false))
		return err
	}

	_, err := c.Write(Encode([]byte(val), false))
	return err
}

func evalDEL(args []string, c net.Conn) error {

	if len(args) < 1 {
		return errors.New("ERR wrong number of arguments for 'del' command")
	}

	count := 0
	for _, key := range args {
		if deleteKey(key) {
			count++
		}
	}

	_, err := c.Write(Encode(count, false))
	return err
}

func evalEXPIRE(args []string, c net.Conn) error {

	if len(args) != 2 {
		return errors.New("ERR wrong number of arguments for 'expire' command")
	}

	seconds, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errors.New("ERR value is not an integer or out of range")
	}

	var ok bool
	if seconds <= 0 {
		ok = deleteKey(args[0])
	} else {
		ok = expireKey(args[0], time.Duration(seconds)*time.Second)
	}

	result := 0
	if ok {
		result = 1
	}

	_, werr := c.Write(Encode(result, false))
	return werr
}

func evalEXISTS(args []string, c net.Conn) error {

	if len(args) < 1 {
		return errors.New("ERR wrong number of arguments for 'exists' command")
	}

	count := 0
	for _, key := range args {
		if _, ok := getKey(key); ok {
			count++
		}
	}

	_, err := c.Write(Encode(count, false))
	return err
}

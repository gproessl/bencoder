package bencode

import (
	"math/big"
	"strconv"
	"io/ioutil"
)


//--------------------------------------------------------
// ERROR
//--------------------------------------------------------

type BencodeError struct {
	msg string
}

func (err BencodeError) Error() string {
	return err.msg
}

//--------------------------------------------------------
// ELEMENT INTERFACE
//--------------------------------------------------------
type Element interface {
	String() string
	Encode() []byte
}


//--------------------------------------------------------
// ELEMENT TYPES
//--------------------------------------------------------
type Integer struct {
	Value *big.Int
}

func NewInteger(val int) Integer {
  return Integer { Value: big.NewInt(int64(val)) }
}

type ByteString struct {
	Value []byte
}

func newByteString(val []byte) ByteString {
  return ByteString { Value: val }
}

type List struct {
	Value []Element
}

func newList(val []Element) List {
  return List { Value: val }
}

type Dictionary struct {
	Value [][2]Element
}

func newDictionary(val [][2]Element) Dictionary {
  return Dictionary { Value: val }
}

func (dic Dictionary) Get(val string) (Element, error) {
	var res Element
	for i := range dic.Value {
		if dic.Value[i][0].String() == val {
			return dic.Value[i][1], nil
		}
	}
	return res, BencodeError { msg: "key not present in Dictionary" }
}



//--------------------------------------------------------
// INTERFACE FUNCTIONS
//--------------------------------------------------------

func (elem Integer) String() string {
	return elem.Value.String()
}

func (elem ByteString) String() string {
	res := ""
	for _, val := range elem.Value {
		if val >= 32 && val <= 126 {
			res += string(val)
		} else {
			res += string(".")
		}
	}
	return res
}

func (elem List) String() string {
	res := "{\n"
	for i := range elem.Value {
		res += elem.Value[i].String() + "\n"
	}
	res += "}"
	return res
}

func (elem Dictionary) String() string {
	res := "{\n"
	for i := range elem.Value {
		res += elem.Value[i][0].String() + " => " + elem.Value[i][1].String() + "\n"
	}
	res += "}"
	return res
}

func (elem Integer) Encode() []byte {
	res := append([]byte("i"), []byte(elem.String())...)
	return append(res, []byte("e")...)
}

func (elem ByteString) Encode() []byte {
	res := append([]byte("b"), []byte(strconv.Itoa(len(elem.Value)))...)
	res = append(res, []byte(":")...)
	res = append(res, elem.Value...)
	return append(res, []byte("e")...)
}

func (elem List) Encode() []byte {
	res := []byte("l")
	for i := range elem.Value {
		res = append(res, elem.Value[i].Encode()...)
	}
	return append(res, []byte("e")...)
}

func (elem Dictionary) Encode() []byte {
	res := []byte("d")
	for i := range elem.Value {
		res = append(res, elem.Value[i][0].Encode()...)
		res = append(res, elem.Value[i][1].Encode()...)
	}
	return append(res, []byte("e")...)
}

func FromFile(file string) (Element, error) {
    dat, err := ioutil.ReadFile(file)
	var res Element
	if err != nil {
		return res, BencodeError { msg: "unable to read from file" }
	}
	return D(dat)
}

func D(data []byte) (Element, error) {
	res, _, err := Decode(data)
	if err != nil {
		return res, err
	}
	return res, nil
}

func Decode(data []byte) (Element, int, error) {
	if len(data) <= 1 {
		return nil, 0, BencodeError{ msg: "missing data" }
	}
	switch data[0] {
	case 'i':
		return decodeInteger(data)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return decodeByteString(data)
	case 'l':
		return decodeList(data)
	case 'd':
		return decodeDictionary(data)
	default:
		return nil, 0, BencodeError{ msg: "inValueid format"}
	}
}

func getEnd(data []byte) (int, error) {
	for i := 0; i < len(data); i++ {
		if data[i] == 'e' {
			return i, nil
		}
	}
	return 0, BencodeError{ msg: "missing end" }
}

func decodeInteger(data []byte) (Integer, int, error) {
	var res Integer
	e, err := getEnd(data)
	if err != nil {
		return res, 0, err
	}
	v, err := strconv.Atoi(string(data[1:e]))
	if err != nil {
		return res, 0, err
	}
	res.Value = big.NewInt(int64(v))
	return res, e, nil
}

func decodeByteString(data []byte) (ByteString, int, error) {
	var res ByteString
	dataSize := len(data)
	if dataSize <= 1 {
		return res,0, BencodeError{ msg: "bytestring does not contain length" }
	}
	var i int
	for i = 1; data[i] != ':'; i++ {
		if i >= dataSize - 1 {
			return res, 0, BencodeError{ msg: "bytestring does not contain length" }
		}
	}
	l, err := strconv.Atoi(string(data[:i]))
	if err != nil {
		return res, 0, BencodeError{ msg: "unable to parse length of bytestring" }
	}
	if dataSize <= i + l {
		return res, 0, BencodeError{ msg: "missing data" }
	}
	if l == 0 {
		res.Value = []byte{}
	} else {
		res.Value = data[i + 1:i + l + 1]
	}
	return res, i + l, nil
}

func decodeList(data []byte) (List, int, error) {
	var res List
	var i int
	for i = 1; data[i] != 'e'; {
		elem, e, err := Decode(data[i:])
		if err != nil {
			return res, 0, err
		}
		res.Value = append(res.Value, elem)
		i += e + 1
		if i >= len(data) {
			return res, 0, BencodeError{ msg: "missing end of list" }
		}
	}
	return res, i, nil
}

func decodeDictionary(data []byte) (Dictionary, int, error) {
	var res Dictionary
	var i int
	for i = 1; data[i] != 'e'; {
		k, e, err := Decode(data[i:])
		if err != nil {
			return res, 0, err
		}
		i += e + 1
		if i >= len(data) {
			return res, 0, BencodeError{ msg: "missing key in Dictionary" }
		}
		v, e, err := Decode(data[i:])
		if err != nil {
			return res, 0, err
		}

		res.Value = append(res.Value, [2]Element { k, v })
		i += e + 1
		if i >= len(data) {
			return res, 0, BencodeError{ msg: "missing end of Dictionary" }
		}
	}
	return res, i, nil
}

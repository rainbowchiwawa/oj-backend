package utility

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

func decodeChineseString(str string) string {
	if utf8.ValidString(str) {
		return str
	}
	decoder := traditionalchinese.Big5.NewDecoder()
	if decoded, err := decoder.String(str); err == nil && !strings.Contains(decoded, "\uFFFD") {
		return decoded
	}
	decoderGBK := simplifiedchinese.GBK.NewDecoder()
	if decoded, err := decoderGBK.String(str); err == nil && !strings.Contains(decoded, "\uFFFD") {
		return decoded
	}
	return str
}

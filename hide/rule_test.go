package hide

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFullExclude(t *testing.T) {
	expectedVal := "*******"
	outputVal := fullExclude("awesome")
	assert.Equal(t, expectedVal, outputVal)
}

func TestMaskName(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{"Ян", "**"},
		{"Владимир", "Вл******"},
		{"Сан-Себастьян", "Са*-Се*******"},
		{"Владимир Иванович Путилин", "Вл****** Ив****** Пу*****"},
		{"Ким Чи Мин", "Ки* Чи Ми*"},
		{"Mark", "Ma**"},
		{"John Doe", "Jo** Do*"},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskName(tt.name))
		})
	}
}

func TestMaskPhone(t *testing.T) {
	testCases := []struct {
		phone    string
		expected string
	}{
		{"79996663311", "*******3311"},
		{"+7 999 666 3311", "+* *** *** 3311"},
		{"7-(999)-666-33-11", "*-(***)-***-*3-11"},
	}
	for _, tt := range testCases {
		t.Run(tt.phone, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskCardAndPhone(tt.phone))
		})
	}
}

func TestMaskCard(t *testing.T) {
	testCases := []struct {
		phone    string
		expected string
	}{
		{"4261 0000 5555 4444", "**** **** **** 4444"},
		{"4261000055554444", "************4444"},
	}
	for _, tt := range testCases {
		t.Run(tt.phone, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskCardAndPhone(tt.phone))
		})
	}
}

func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		email    string
		expected string
	}{
		{"normal@email.com", "nor***@email.com"},
		{"a@custom.site", "*@custom.site"},
		{"site.com", "********"},
	}
	for _, tt := range testCases {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskEmail(tt.email))
		})
	}
}

func TestMaskURL(t *testing.T) {
	setTestConvertor()
	testCases := []struct {
		url      string
		expected string
	}{
		{"https://site.com/path?token=secret&id=2", "https://site.com/path?id=2&token=******"},
		{"site.com?id=2&token=secret", "site.com?id=2&token=******"},
		{"id=2&token=secret&lastname=Василий%20Петрович", "?id=2&lastname=Ва***** Пе******&token=******"},
		{"https://username:password@site.com/path?token=secret", "https://username:xxxxx@site.com/path?token=******"},
		{"?id=2&password=awesome&ref=head", "?id=2&password=*******&ref=head"},
		{"http://site.com", "http://site.com"},
		{"?", "?"},
	}
	for _, tt := range testCases {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskURL(tt.url))
		})
	}
}

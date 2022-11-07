package internal

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAppendAccName(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "Name | Account Name.1234"
	name := "Name"
	accName := "Account Name.1234"
	nick := appendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "Veri | fy Long Account Name.1234"
	name := "Verify Long Discord User Name!!!!!!!!!!!!"
	accName := "Verify Long Account Name.1234"
	nick := appendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameShortNickLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "3rd | ify Long Account Name.1234"
	name := "3rd"
	accName := "Verify Verify Verify Long Account Name.1234"
	nick := appendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameExistingAccNameNick(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "3rd | ify Long Account Name.1234"
	name := "3rd | ify Long Account Name.1234"
	accName := "Verify Verify Verify Long Account Name.1234"
	nick := appendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

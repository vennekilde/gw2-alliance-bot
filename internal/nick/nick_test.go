package nick

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAppendAccName(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "Name | Account Name.1234"
	name := "Name"
	accName := "Account Name.1234"
	nick := AppendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "Veri | fy Long Account Name.1234"
	name := "Verify Long Discord User Name!!!!!!!!!!!!"
	accName := "Verify Long Account Name.1234"
	nick := AppendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameShortNickLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "3rd | ify Long Account Name.1234"
	name := "3rd"
	accName := "Verify Verify Verify Long Account Name.1234"
	nick := AppendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestAppendAccNameExistingAccNameNick(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "3rd | ify Long Account Name.1234"
	name := "3rd | ify Long Account Name.1234"
	accName := "Verify Verify Verify Long Account Name.1234"
	nick := AppendAccName(name, accName)
	g.Expect(nick).To(Equal(expected))
}

func TestPrependGuildTag(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "[TEST] Test Name"
	name := "Test Name"
	guildTag := "TEST"
	nick := PrependGuildTag(name, guildTag)
	g.Expect(nick).To(Equal(expected))
}

func TestPrependGuildTagLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "[TES] Verify Long Discord User N"
	name := "Verify Long Discord User Name!!!!!!!!!!!!"
	guildTag := "TES"
	nick := PrependGuildTag(name, guildTag)
	g.Expect(nick).To(Equal(expected))
	g.Expect(len(nick)).To(BeNumerically("<=", maxDiscordNickLen))
}

func TestPrependGuildTagExistingAccNameNick(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "[TES] Verify Long Discord User N"
	name := "[DW] Verify Long Discord User Name!!!!!!!!!!!!"
	guildTag := "TES"
	nick := PrependGuildTag(name, guildTag)
	g.Expect(nick).To(Equal(expected))
	g.Expect(len(nick)).To(BeNumerically("<=", maxDiscordNickLen))
}

func TestPrependGuildTagWithExclamationLong(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "![TES] Verify Long Discord User "
	name := "!Verify Long Discord User Name!!!!!!!!!!!!"
	guildTag := "TES"
	nick := PrependGuildTag(name, guildTag)
	g.Expect(nick).To(Equal(expected))
	g.Expect(len(nick)).To(BeNumerically("<=", maxDiscordNickLen))
}

func TestPrependGuildTagWithExclamationExistingAccNameNick(t *testing.T) {
	g := NewGomegaWithT(t)
	expected := "![TES] Verify Long Discord User "
	name := "![DW] Verify Long Discord User Name!!!!!!!!!!!!"
	guildTag := "TES"
	nick := PrependGuildTag(name, guildTag)
	g.Expect(nick).To(Equal(expected))
	g.Expect(len(nick)).To(BeNumerically("<=", maxDiscordNickLen))
}

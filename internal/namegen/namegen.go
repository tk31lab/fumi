// Package namegen generates container names themed on Japanese gardens,
// e.g. "fumi-shizuka-tsuki" (a quiet moon).
package namegen

import "math/rand/v2"

// Prefix is prepended to every generated name so that fumi's containers are
// easy to find with `docker ps --filter name=fumi-`.
const Prefix = "fumi"

// adjectives describe a mood or appearance (romanized Japanese).
var adjectives = []string{
	"shizuka",  // 静か  quiet
	"hisoka",   // 密か  secret
	"oboro",    // 朧    hazy
	"haruka",   // 遥か  distant
	"nodoka",   // 長閑  tranquil
	"kasuka",   // 幽か  faint
	"sayaka",   // 清か  clear
	"hanayaka", // 華やか brilliant
	"yuruyaka", // 緩やか gentle
	"odayaka",  // 穏やか calm
	"sumiyaka", // 速やか swift
	"yawaraka", // 柔らか soft
	"ao",       // 青    blue
	"shiro",    // 白    white
	"kuro",     // 黒    black
	"aka",      // 赤    red
	"kin",      // 金    golden
	"gin",      // 銀    silver
	"fukai",    // 深い  deep
	"furui",    // 古い  ancient
}

// nouns are things found in a Japanese garden.
var nouns = []string{
	"ishi",     // 石   stone
	"koke",     // 苔   moss
	"kaede",    // 楓   maple
	"momiji",   // 紅葉 autumn leaves
	"sakura",   // 桜   cherry blossom
	"take",     // 竹   bamboo
	"matsu",    // 松   pine
	"ume",      // 梅   plum
	"tsuki",    // 月   moon
	"kiri",     // 霧   mist
	"kaze",     // 風   wind
	"mizu",     // 水   water
	"ike",      // 池   pond
	"taki",     // 滝   waterfall
	"hashi",    // 橋   bridge
	"tourou",   // 灯籠 stone lantern
	"sekitei",  // 石庭 rock garden
	"hotaru",   // 蛍   firefly
	"suna",     // 砂   sand
	"tobiishi", // 飛石 stepping stone
}

// Generate returns a random name of the form "fumi-<adjective>-<noun>".
func Generate() string {
	return generate(rand.IntN)
}

func generate(intn func(int) int) string {
	return Prefix + "-" + adjectives[intn(len(adjectives))] + "-" + nouns[intn(len(nouns))]
}

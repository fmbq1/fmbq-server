package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// AvatarColors represents the available avatar background colors
var AvatarColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7",
	"#DDA0DD", "#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E9",
	"#F8C471", "#82E0AA", "#F1948A", "#85C1E9", "#D7BDE2",
	"#A9DFBF", "#F9E79F", "#D5DBDB", "#AED6F1", "#FADBD8",
}

// AvatarShapes represents the available avatar shapes
var AvatarShapes = []string{
	"circle", "square", "rounded", "diamond",
}

// GenerateRandomAvatar generates a random avatar URL using a service like DiceBear
func GenerateRandomAvatar() string {
	// Generate random seed for consistent avatars
	seed, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	
	// Use DiceBear API with random style and seed
	styles := []string{"avataaars", "personas", "micah", "miniavs", "bottts"}
	styleIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(styles))))
	style := styles[styleIndex.Int64()]
	
	return fmt.Sprintf("https://api.dicebear.com/7.x/%s/svg?seed=%d", style, seed.Int64())
}

// GenerateAvatarWithInitials generates an avatar with user initials
func GenerateAvatarWithInitials(initials string) string {
	// Generate random color
	colorIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(AvatarColors))))
	color := AvatarColors[colorIndex.Int64()]
	
	// Generate random shape
	shapeIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(AvatarShapes))))
	shape := AvatarShapes[shapeIndex.Int64()]
	
	// Use DiceBear API with initials
	return fmt.Sprintf("https://api.dicebear.com/7.x/initials/svg?seed=%s&backgroundColor=%s&shape=%s", 
		initials, color, shape)
}

// GetInitialsFromName extracts initials from a full name
func GetInitialsFromName(name string) string {
	if name == "" {
		return "U"
	}
	
	words := []rune(name)
	initials := ""
	
	// Get first character
	initials += string(words[0])
	
	// Find space and get next character
	for i, char := range words {
		if char == ' ' && i+1 < len(words) {
			initials += string(words[i+1])
			break
		}
	}
	
	// If only one character, duplicate it
	if len(initials) == 1 {
		initials += initials
	}
	
	return initials
}

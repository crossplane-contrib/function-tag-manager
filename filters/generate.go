//go:build generate

package filters

// Regenerate the resource filters from the provider versions pinned in
// versions.env. See generate.sh.
//go:generate sh ./generate.sh

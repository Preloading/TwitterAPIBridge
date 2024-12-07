package twitterv1

import (
	"bytes"
	"encoding/xml"
	"fmt"

	blueskyapi "github.com/Preloading/MastodonTwitterAPI/bluesky"
	"github.com/gofiber/fiber/v2"
)

// https://web.archive.org/web/20120508075505/https://dev.twitter.com/docs/api/1/get/users/show
func user_info(c *fiber.Ctx) error {
	screen_name := c.Query("screen_name")
	fmt.Println("screen_name:", screen_name)
	_, oauthToken, err := GetAuthFromReq(c)

	if err != nil {
		blankstring := ""
		oauthToken = &blankstring
	}

	userinfo, err := blueskyapi.GetUserInfo(*oauthToken, screen_name)

	if err != nil {
		fmt.Println("Error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch user info")
	}

	// TODO: Clean this up a bit, see if there's an easier way

	// Encode the userinfo to XML
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(userinfo); err != nil {
		fmt.Println("Error encoding XML:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to encode user info")
	}

	// Remove the root element <TwitterUserWithStatus> and replace with custom header
	xmlContent := buf.Bytes()
	start := bytes.Index(xmlContent, []byte("<TwitterUserWithStatus>"))
	end := bytes.LastIndex(xmlContent, []byte("</TwitterUserWithStatus>"))
	if start == -1 || end == -1 {
		return c.Status(fiber.StatusInternalServerError).SendString("Invalid XML format")
	}
	xmlContent = xmlContent[start+len("<TwitterUserWithStatus>") : end]

	// Add custom XML header and root element
	customHeader := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<user>` + "\n")
	xmlContent = append(customHeader, xmlContent...)

	// Add custom footer
	customFooter := []byte("\n</user>")
	xmlContent = append(xmlContent, customFooter...)

	return c.Send(xmlContent)
}

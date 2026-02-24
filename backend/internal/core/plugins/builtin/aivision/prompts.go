package aivision

import (
	"fmt"
	"strings"
)

// BuildStep1Prompt creates the vision model prompt for hardware identification.
func BuildStep1Prompt(imageIndex, totalImages int, contextItems []string, userContext string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert hardware identification system. Analyze this photo and identify all distinct items.\n\n")

	sb.WriteString("For each item, extract:\n")
	sb.WriteString("- name: Specific product name (e.g., 'Supermicro X9DRi-F Motherboard', not just 'motherboard')\n")
	sb.WriteString("- quantity: How many of this exact item (default 1)\n")
	sb.WriteString("- serial_number: Any visible serial numbers\n")
	sb.WriteString("- part_number: Model/part numbers from silk-screen or labels\n")
	sb.WriteString("- description: Brief description including specs visible in photo\n")
	sb.WriteString("- estimated_category: One of: motherboard, cpu, ram, gpu, storage, nic, psu, case, cable, cooling, peripheral, other\n")
	sb.WriteString("- label_text: Key text from labels/stickers for record-keeping\n\n")

	sb.WriteString("Focus on reading:\n")
	sb.WriteString("- Silk-screen text on PCBs\n")
	sb.WriteString("- Serial number stickers\n")
	sb.WriteString("- MAC addresses\n")
	sb.WriteString("- IPMI/BMC addresses\n")
	sb.WriteString("- SAS/storage controller addresses\n")
	sb.WriteString("- RAM module labels (capacity, speed, ECC status)\n")
	sb.WriteString("- Model numbers on any component\n\n")

	if len(contextItems) > 0 {
		sb.WriteString(fmt.Sprintf("Previously identified items (DO NOT re-identify these): %s\n\n", strings.Join(contextItems, ", ")))
	}

	if userContext != "" {
		sb.WriteString(fmt.Sprintf("User context: %s\n\n", userContext))
	}

	sb.WriteString(fmt.Sprintf("This is image %d of %d.\n\n", imageIndex+1, totalImages))
	sb.WriteString("Return a JSON array of identified items. If no items are identifiable, return [].")

	return sb.String()
}

// BuildStep2Prompt creates the verification prompt for Gemini (text-only).
func BuildStep2Prompt(rawItemsJSON string, locations []string, labels []string, userContext string) string {
	var sb strings.Builder

	sb.WriteString("You are a hardware verification and inventory classification system.\n\n")
	sb.WriteString("Review the following items identified by a vision AI and verify/enhance each entry:\n\n")
	sb.WriteString("Raw items from vision model:\n")
	sb.WriteString(rawItemsJSON)
	sb.WriteString("\n\n")

	sb.WriteString("For each item, verify and enhance:\n")
	sb.WriteString("1. Validate server vs. consumer classification (ECC RAM = server, non-ECC = consumer)\n")
	sb.WriteString("2. Check RAM specs: verify total capacity = quantity x per-stick capacity\n")
	sb.WriteString("3. Validate CPU-socket pairings (e.g., Xeon E5-2600 v2 = LGA 2011)\n")
	sb.WriteString("4. Add confidence score (0-100) for each item\n")
	sb.WriteString("5. Suggest the best storage location from the available locations\n")
	sb.WriteString("6. Suggest appropriate labels/tags\n")
	sb.WriteString("7. Add any correction notes\n\n")

	if len(locations) > 0 {
		sb.WriteString("Available storage locations:\n")
		for _, loc := range locations {
			sb.WriteString(fmt.Sprintf("- %s\n", loc))
		}
		sb.WriteString("\n")
	}

	if len(labels) > 0 {
		sb.WriteString("Available labels/tags:\n")
		for _, label := range labels {
			sb.WriteString(fmt.Sprintf("- %s\n", label))
		}
		sb.WriteString("\n")
	}

	if userContext != "" {
		sb.WriteString(fmt.Sprintf("User context: %s\n\n", userContext))
	}

	sb.WriteString("Return a JSON array with these fields per item:\n")
	sb.WriteString("name, quantity, serial_number, part_number, description, suggested_location, suggested_labels, label_text, confidence, notes\n\n")
	sb.WriteString("If an item seems incorrect or hallucinated, set confidence to 0 and explain in notes.")

	return sb.String()
}

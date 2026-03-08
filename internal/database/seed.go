package database

import (
	"fmt"

	"gorm.io/gorm"
)

func SeedDatabase(db *gorm.DB) error {
	// 1. Define 5 Categories
	categories := []Category{
		{Name: "Electronics"},    // Index 0
		{Name: "Home & Kitchen"}, // Index 1
		{Name: "Stationery"},     // Index 2
		{Name: "Fitness"},        // Index 3
		{Name: "Personal Care"},  // Index 4
	}

	for i := range categories {
		err := db.Where(Category{Name: categories[i].Name}).FirstOrCreate(&categories[i]).Error
		if err != nil {
			return fmt.Errorf("could not seed category %s: %v", categories[i].Name, err)
		}
	}

	// 2. Define 25 Products (5 per category)
	products := []Product{
		// --- Electronics ---
		{Name: "Mechanical Keyboard", Description: "RGB Backlit Blue Switches", Price: 89.99, Available: 50, ImageObjKey: "products/keyboard.jpg", CategoryID: categories[0].ID},
		{Name: "Gaming Mouse", Description: "16000 DPI Optical Sensor", Price: 45.50, Available: 80, ImageObjKey: "products/mouse.jpg", CategoryID: categories[0].ID},
		{Name: "Noise Cancelling Headphones", Description: "Wireless Over-ear", Price: 199.00, Available: 30, ImageObjKey: "products/headphones.jpg", CategoryID: categories[0].ID},
		{Name: "USB-C Hub", Description: "7-in-1 Aluminum Adapter", Price: 35.00, Available: 120, ImageObjKey: "products/hub.jpg", CategoryID: categories[0].ID},
		{Name: "Monitor Stand", Description: "Adjustable height with drawer", Price: 29.99, Available: 40, ImageObjKey: "products/stand.jpg", CategoryID: categories[0].ID},

		// --- Home & Kitchen ---
		{Name: "Air Fryer", Description: "5L Digital Touch Screen", Price: 120.50, Available: 15, ImageObjKey: "products/airfryer.jpg", CategoryID: categories[1].ID},
		{Name: "Electric Kettle", Description: "1.7L Stainless Steel", Price: 34.99, Available: 60, ImageObjKey: "products/kettle.jpg", CategoryID: categories[1].ID},
		{Name: "French Press", Description: "1L Glass Coffee Maker", Price: 18.00, Available: 45, ImageObjKey: "products/frenchpress.jpg", CategoryID: categories[1].ID},
		{Name: "Knife Block Set", Description: "15-piece Professional Steel", Price: 75.00, Available: 10, ImageObjKey: "products/knives.jpg", CategoryID: categories[1].ID},
		{Name: "Digital Scale", Description: "High precision kitchen scale", Price: 12.99, Available: 150, ImageObjKey: "products/scale.jpg", CategoryID: categories[1].ID},

		// --- Stationery ---
		{Name: "Fountain Pen", Description: "Fine nib classic pen", Price: 25.00, Available: 100, ImageObjKey: "products/pen.jpg", CategoryID: categories[2].ID},
		{Name: "Hardcover Notebook", Description: "A5 Dotted Paper", Price: 15.99, Available: 200, ImageObjKey: "products/notebook.jpg", CategoryID: categories[2].ID},
		{Name: "Desk Organizer", Description: "Mesh Metal 6-compartment", Price: 19.50, Available: 55, ImageObjKey: "products/organizer.jpg", CategoryID: categories[2].ID},
		{Name: "Gel Pen Set", Description: "12 Colors Fine Point", Price: 14.00, Available: 90, ImageObjKey: "products/gelpens.jpg", CategoryID: categories[2].ID},
		{Name: "Sticky Notes Bundle", Description: "Assorted neon colors", Price: 8.50, Available: 300, ImageObjKey: "products/stickynotes.jpg", CategoryID: categories[2].ID},

		// --- Fitness ---
		{Name: "Yoga Mat", Description: "6mm Thick Non-slip", Price: 22.00, Available: 75, ImageObjKey: "products/yogamat.jpg", CategoryID: categories[3].ID},
		{Name: "Dumbbell Set", Description: "5lb - 20lb Pair", Price: 55.00, Available: 20, ImageObjKey: "products/dumbbells.jpg", CategoryID: categories[3].ID},
		{Name: "Resistance Bands", Description: "Set of 5 levels", Price: 12.50, Available: 110, ImageObjKey: "products/bands.jpg", CategoryID: categories[3].ID},
		{Name: "Jump Rope", Description: "Speed rope with bearings", Price: 9.99, Available: 140, ImageObjKey: "products/jumprope.jpg", CategoryID: categories[3].ID},
		{Name: "Water Bottle", Description: "1L Vacuum Insulated", Price: 24.00, Available: 95, ImageObjKey: "products/bottle.jpg", CategoryID: categories[3].ID},

		// --- Personal Care ---
		{Name: "Electric Toothbrush", Description: "Sonic vibration with 3 modes", Price: 49.99, Available: 40, ImageObjKey: "products/toothbrush.jpg", CategoryID: categories[4].ID},
		{Name: "Beard Trimmer", Description: "Cordless with 20 settings", Price: 38.00, Available: 35, ImageObjKey: "products/trimmer.jpg", CategoryID: categories[4].ID},
		{Name: "Hair Dryer", Description: "1800W Ionic Pro", Price: 59.00, Available: 25, ImageObjKey: "products/hairdryer.jpg", CategoryID: categories[4].ID},
		{Name: "Face Cleanser", Description: "Organic Aloe Vera 200ml", Price: 16.50, Available: 120, ImageObjKey: "products/cleanser.jpg", CategoryID: categories[4].ID},
		{Name: "Hand Cream", Description: "Shea Butter Repair 50ml", Price: 7.99, Available: 200, ImageObjKey: "products/handcream.jpg", CategoryID: categories[4].ID},
	}

	for _, prod := range products {
		err := db.Where(Product{Name: prod.Name}).FirstOrCreate(&prod).Error
		if err != nil {
			return fmt.Errorf("could not seed product %s: %v", prod.Name, err)
		}
	}

	fmt.Println("✅ Database seeded with 5 categories and 25 products!")
	return nil
}

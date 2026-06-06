package database

import (
	"context"
	"fmt"
	"lorem-backend/internal/utils"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeedFileRepository interface {
	UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error)
	CreateFileMeta(ctx context.Context, fileMeta *File) (uuid.UUID, error)
}

type seedData struct {
	CategoryName string
	Name         string
	Description  string
	Price        float32
	Available    uint
	ImageFile    string
}

func SeedDatabase(ctx context.Context, db *gorm.DB, fileRepo SeedFileRepository) error {
	// 1. Define 2 Categories
	categories := []Category{
		{Name: "Apparel"},   // Index 0
		{Name: "Accessory"}, // Index 1
	}

	categoryMap := make(map[string]uuid.UUID)

	for i := range categories {
		err := db.Where(Category{Name: categories[i].Name}).FirstOrCreate(&categories[i]).Error
		if err != nil {
			return fmt.Errorf("could not seed category %s: %v", categories[i].Name, err)
		}
		categoryMap[categories[i].Name] = categories[i].ID
	}

	// 2. Define Products
	productsData := []seedData{
		// --- Apparel ---
		{"Apparel", "Classic Cotton T-Shirt", "100% Organic Cotton, Crew Neck", 19.99, 200, "apparel/t-shirt.jpg"},
		{"Apparel", "Fleece Pullover Hoodie", "Heavyweight warm fleece", 49.50, 85, "apparel/hoodie.jpg"},
		{"Apparel", "Slim Fit Jeans", "Stretch denim, dark wash", 59.99, 120, "apparel/jeans.jpg"},
		{"Apparel", "Denim Jacket", "Vintage wash trucker jacket", 75.00, 40, "apparel/denim-jacket.jpg"},
		{"Apparel", "Knit Crewneck Sweater", "Merino wool blend", 65.00, 60, "apparel/sweater.jpg"},
		{"Apparel", "Athletic Shorts", "Breathable mesh with pockets", 25.00, 150, "apparel/athletic-shorts.jpg"},
		{"Apparel", "Polo Shirt", "Pique cotton short sleeve", 35.00, 110, "apparel/polo-shirt.jpg"},
		{"Apparel", "Jogger Sweatpants", "Tapered fit with drawstring", 39.99, 95, "apparel/jogger-sweatpants.jpg"},

		// --- Accessory ---
		{"Accessory", "Minimalist Wristwatch", "Stainless steel mesh band, quartz", 125.00, 40, "accessories/watch.jpg"},
		{"Accessory", "Aviator Sunglasses", "Polarized lenses with UV400", 85.00, 65, "accessories/sunglasses.jpg"},
		{"Accessory", "Classic Leather Belt", "Full-grain leather, silver buckle", 34.99, 100, "accessories/belt.jpg"},
		{"Accessory", "Knit Beanie", "Warm winter hat, unisex", 18.00, 150, "accessories/knit-beanie.jpg"},
		{"Accessory", "Canvas Baseball Cap", "Adjustable strap, embroidered logo", 22.50, 120, "accessories/cap.jpg"},
		{"Accessory", "Cashmere Scarf", "Ultra-soft winter neckwear", 60.00, 45, "accessories/scarf.jpg"},
		{"Accessory", "Bifold Leather Wallet", "RFID blocking with coin pocket", 45.00, 80, "accessories/wallet.jpg"},
		{"Accessory", "Everyday Backpack", "Water-resistant with laptop sleeve", 75.00, 55, "accessories/bagpack.jpg"},
		{"Accessory", "Canvas Messenger Bag", "Vintage style crossbody bag", 65.50, 35, "accessories/bagpack.jpg"},
		{"Accessory", "Leather Gloves", "Touchscreen compatible, fleece-lined", 48.00, 60, "accessories/gloves.jpg"},
		{"Accessory", "Silk Necktie", "Textured weave, standard width", 28.00, 90, "accessories/necktie.jpg"},
		{"Accessory", "Gold Earrings", "14k Gold plated minimalist hoops", 35.00, 75, "accessories/gold-earrings.jpg"},
		{"Accessory", "Pendant Necklace", "Sterling silver chain with geometric pendant", 42.00, 50, "accessories/gold-necklace.jpg"},
		{"Accessory", "Braided Leather Bracelet", "Magnetic steel clasp", 24.00, 110, "accessories/leather-bracelet.jpg"},
	}

	for _, p := range productsData {
		var existingProduct Product
		err := db.Where(Product{Name: p.Name}).First(&existingProduct).Error
		if err == nil {
			// Product already exists, skip
			continue
		}

		// 3. Upload Image
		imagePath := filepath.Join("..", "static", filepath.FromSlash(p.ImageFile))
		f, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("could not open image %s: %v", imagePath, err)
		}

		stat, _ := f.Stat()
		fileName := filepath.Base(p.ImageFile)
		putKey := fmt.Sprintf("product-images/%v-%v", time.Now().UnixNano(), fileName)

		// Upload to Object Storage
		objKey, err := fileRepo.UploadFile(ctx, putKey, f, stat.Size(), "image/jpeg")
		f.Close()
		if err != nil {
			return fmt.Errorf("error uploading file to S3: %v", err)
		}

		// Store File Metadata to database
		fileMeta := &File{
			OriginalName: fileName,
			Name:         uuid.New().String(),
			Size:         stat.Size(),
			ContentType:  "image/jpeg",
			ObjectKey:    objKey,
		}

		_, err = fileRepo.CreateFileMeta(ctx, fileMeta)
		if err != nil {
			return fmt.Errorf("error generating file metadata: %v", err)
		}

		product := Product{
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Available:   p.Available,
			ImageObjKey: objKey,
			CategoryID:  categoryMap[p.CategoryName],
		}

		err = db.Create(&product).Error
		if err != nil {
			return fmt.Errorf("could not seed product %s: %v", p.Name, err)
		}
	}

	// 3. Add Static Files
	staticImages := []string{
		"auth-banner.jpg",
		"hero/hero.jpg",
		"hero-sm.jpg",
		"home-apparel.jpg",
		"home-accessory.jpg",
		"apparelSlide1.jpg",
		"apparelSlide2.jpg",
		"apparelSlide3.jpg",
		"accessorySlide1.jpg",
		"accessorySlide2.jpg",
		"accessorySlide3.jpg",
	}

	for _, imgName := range staticImages {
		imagePath := filepath.Join("..", "static", filepath.FromSlash(imgName))
		f, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("could not open image %s: %v", imagePath, err)
		}

		stat, _ := f.Stat()
		fileName := filepath.Base(imgName)
		putKey := fmt.Sprintf("static/%v", fileName)

		// Upload to Object Storage
		objKey, err := fileRepo.UploadFile(ctx, putKey, f, stat.Size(), "image/jpeg")
		f.Close()
		if err != nil {
			return fmt.Errorf("error uploading file to S3: %v\n", err)
		}

		fmt.Printf("The image is stored at %v\n", objKey)
	}

	// 4. Define test user
	hashed, _ := utils.HashPassword("password123")
	isNotAdmin := false
	telephone := "0812345678"
	zipCode := "10110"
	road := "Sukhumvit Rd"
	district := "Watthana"
	subDistrict := "Khlong Toei Nuea"
	houseNumber := "123/45"
	province := "Bangkok"

	testUser := User{
		Username:     "testuser",
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "testuser@example.com",
		PasswordHash: hashed,
		IsAdmin:      &isNotAdmin,
		Telephone:    &telephone,
		ZipCode:      &zipCode,
		Road:         &road,
		District:     &district,
		SubDistrict:  &subDistrict,
		HouseNumber:  &houseNumber,
		Province:     &province,
	}

	if err := db.Where(User{Email: "testuser@example.com"}).FirstOrCreate(&testUser).Error; err != nil {
		return fmt.Errorf("could not seed test user: %v", err)
	}
	testUser.Username = "testuser"
	testUser.FirstName = "John"
	testUser.LastName = "Doe"
	testUser.PasswordHash = hashed
	testUser.IsAdmin = &isNotAdmin
	testUser.Telephone = &telephone
	testUser.ZipCode = &zipCode
	testUser.Road = &road
	testUser.District = &district
	testUser.SubDistrict = &subDistrict
	testUser.HouseNumber = &houseNumber
	testUser.Province = &province
	if err := db.Save(&testUser).Error; err != nil {
		return fmt.Errorf("could not update test user: %v", err)
	}

	// Ensure the test user has a cart
	var testUserCart Cart
	if err := db.Where(Cart{UserID: testUser.ID}).FirstOrCreate(&testUserCart).Error; err != nil {
		return fmt.Errorf("could not seed test user cart: %v", err)
	}

	// 5. Define admin user
	isAdmin := true
	adminUser := User{
		Username:     "adminUser",
		FirstName:    "Admin",
		LastName:     "User",
		Email:        "admin@example.com",
		PasswordHash: hashed,
		IsAdmin:      &isAdmin,
	}

	if err := db.Where(User{Email: "admin@example.com"}).FirstOrCreate(&adminUser).Error; err != nil {
		return fmt.Errorf("could not seed admin user: %v", err)
	}
	adminUser.Username = "adminUser"
	adminUser.FirstName = "Admin"
	adminUser.LastName = "User"
	adminUser.PasswordHash = hashed
	adminUser.IsAdmin = &isAdmin
	if err := db.Save(&adminUser).Error; err != nil {
		return fmt.Errorf("could not update admin user: %v", err)
	}

	fmt.Println("✅ Database seeded with 2 categories, 30 products, and 2 users!")
	return nil
}

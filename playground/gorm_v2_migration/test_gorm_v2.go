// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is a minimal proof-of-concept (PoC) to verify that GORM v2's basic APIs
// (Open, AutoMigrate, Create, Find, Update, Delete) work correctly with MySQL.
// The goal is to ensure functional compatibility before integrating GORM v2 into the main codebase.

package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Define model
type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string
	Email string
}


type ResourceReference struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	ResourceUUID    string `gorm:"column:ResourceUUID"`
	ResourceType    string `gorm:"column:ResourceType"`
	ReferenceUUID   string `gorm:"column:ReferenceUUID"`
	ReferenceType   string `gorm:"column:ReferenceType"`
	Relationship    string `gorm:"column:Relationship"`
	CreatedAtInSec  int64  `gorm:"column:CreatedAtInSec"`
	UpdatedAtInSec  int64  `gorm:"column:UpdatedAtInSec"`
}

func main() {
	// Initialize GORM v2 with MySQL
	dsn := "testuser:testpw@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto-migrate table schema
	if err := db.AutoMigrate(&User{}); err != nil {
		panic("failed to migrate")
	}
	if err := db.AutoMigrate(&ResourceReference{}); err != nil {
		panic("failed to migrate ResourceReference")
	}
	fmt.Println("ResourceReference AutoMigrate successful.")

	// Insert data
	db.Create(&User{Name: "Alice", Email: "alice@example.com"})

	// Query data
	var user User
	db.First(&user, "name = ?", "Alice")
	fmt.Printf("User found: %+v\n", user)

	// Find all users
	var users []User
	db.Find(&users)
	fmt.Printf("All users: %+v\n", users)

	// Update user email
	db.Model(&user).Update("Email", "alice_new@example.com")
	fmt.Printf("Updated user: %+v\n", user)

	// Delete user
	db.Delete(&user)
	fmt.Println("User deleted.")

	// Verify deletion
	var count int64
	db.Model(&User{}).Count(&count)
	fmt.Printf("Remaining users count: %d\n", count)

	// Raw SQL: Show tables in MySQL
	var tableNames []string
	db.Raw("SHOW TABLES").Scan(&tableNames)
	fmt.Printf("Tables in DB (via Raw): %v\n", tableNames)

}
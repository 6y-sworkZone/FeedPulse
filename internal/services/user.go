package services

import (
	"database/sql"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/models"
	"feedpulse/internal/utils"
)

func RegisterUser(username, email, password string) (*models.User, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	if err != nil {
		return nil, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = database.DB.Exec(
		"INSERT INTO user_settings (user_id) VALUES (?)",
		userID,
	)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:        userID,
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func LoginUser(email, password string) (*models.User, error) {
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if !utils.CheckPassword(password, user.Password) {
		return nil, sql.ErrNoRows
	}

	return &user, nil
}

func GetUserByID(userID int64) (*models.User, error) {
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, email, created_at, updated_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserSettings(userID int64) (*models.UserSettings, error) {
	var settings models.UserSettings
	err := database.DB.QueryRow(
		"SELECT id, user_id, theme, font_size, articles_per_page, reading_preferences FROM user_settings WHERE user_id = ?",
		userID,
	).Scan(&settings.ID, &settings.UserID, &settings.Theme, &settings.FontSize, &settings.ArticlesPerPage, &settings.ReadingPreferences)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func UpdateUserSettings(userID int64, theme, fontSize string, articlesPerPage int, readingPrefs string) (*models.UserSettings, error) {
	_, err := database.DB.Exec(
		"UPDATE user_settings SET theme = ?, font_size = ?, articles_per_page = ?, reading_preferences = ? WHERE user_id = ?",
		theme, fontSize, articlesPerPage, readingPrefs, userID,
	)
	if err != nil {
		return nil, err
	}
	return GetUserSettings(userID)
}

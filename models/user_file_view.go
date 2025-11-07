package models

// UserFileView представление для отображения в веб-интерфейсе
type UserFileView struct {
	UserID        int64  `json:"user_id"`
	CitizenshipID string `json:"citizenship_id"`
	Document      bool   `json:"document"`
	Address       bool   `json:"address"`
	DocumentFiles string `json:"document_files"` // ссылка из основной БД
	AddressFiles  string `json:"address_files"`  // ссылка из основной БД
}

// PaginatedResponse ответ с пагинацией
type PaginatedResponse struct {
	Data       []UserFileView `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PerPage    int            `json:"per_page"`
	TotalPages int            `json:"total_pages"`
	SortOrder  string         `json:"sort_order"`
}

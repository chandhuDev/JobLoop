package service

import "github.com/chandhuDev/JobLoop/internal/models"

type UtilsService struct {
	NamesChan *models.NamesClient
}

func CreateNamesChannel(bufferSize int) *UtilsService {
	return &UtilsService{
		NamesChan: &models.NamesClient{
			NamesChan: make(chan string, bufferSize),
		},
	}

}

func (u *UtilsService) ReturnNamesChan() *models.NamesClient {
	return u.NamesChan
}

func (u *UtilsService) CloseNamesChan() {
	close(u.NamesChan.NamesChan)
}

package service

import (
	"fmt"
	"strconv"
	"strings"

	"commenttree/internal/models"
	"commenttree/internal/repository"
)

type Service struct {
	repo *repository.Repo
}

func New(repo *repository.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(body, author string, parentID *int64) (*models.Comment, error) {
	if body == "" {
		return nil, fmt.Errorf("body is empty")
	}
	if author == "" {
		author = "Anonymous"
	}
	return s.repo.Create(body, author, parentID)
}

func (s *Service) GetTree(rootID int64) (*models.Comment, error) {
	list, err := s.repo.GetTree(rootID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("not found")
	}
	return buildTree(list), nil
}

func (s *Service) GetRoots(page, limit int) ([]*models.Comment, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	roots, err := s.repo.GetRoots(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountRoots()
	if err != nil {
		return nil, 0, err
	}

	var result []*models.Comment
	for _, root := range roots {
		tree, err := s.GetTree(root.ID)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, tree)
	}

	return result, total, nil
}

func (s *Service) Delete(id int64) error {
	return s.repo.DeleteTree(id)
}

func (s *Service) Search(query string) ([]*models.Comment, error) {
	found, err := s.repo.Search(query)
	if err != nil {
		return nil, err
	}

	rootSet := make(map[int64]bool)
	for _, c := range found {
		parts := strings.Split(strings.Trim(c.Path, "/"), "/")
		if len(parts) > 0 {
			rootID, _ := strconv.ParseInt(parts[0], 10, 64)
			rootSet[rootID] = true
		}
	}

	matchedIDs := make(map[int64]bool)
	for _, c := range found {
		matchedIDs[c.ID] = true
	}

	var result []*models.Comment
	for rootID := range rootSet {
		tree, err := s.GetTree(rootID)
		if err != nil {
			continue
		}
		markMatched(tree, matchedIDs)
		result = append(result, tree)
	}

	return result, nil
}

func buildTree(list []*models.Comment) *models.Comment {
	if len(list) == 0 {
		return nil
	}

	nodeMap := make(map[int64]*models.Comment)
	var root *models.Comment

	for _, c := range list {
		c.Level = strings.Count(c.Path, "/") - 1
		nodeMap[c.ID] = c
	}

	for _, c := range list {
		if c.ParentID == nil {
			root = c
		} else {
			parent, ok := nodeMap[*c.ParentID]
			if ok {
				parent.Children = append(parent.Children, c)
			}
		}
	}

	return root
}

func markMatched(node *models.Comment, matchedIDs map[int64]bool) {
	if matchedIDs[node.ID] {
		node.Matched = true
	}
	for _, child := range node.Children {
		markMatched(child, matchedIDs)
	}
}

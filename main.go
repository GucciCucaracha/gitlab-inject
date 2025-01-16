package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type BadgeData struct {
	Name     string `json:"name"`
	LinkURL  string `json:"link_url"`
	ImageURL string `json:"image_url"`
}

// Group отображает структуру групп в gitlab
type Group struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	ParentID int    `json:"parent_id"`
	Path     string `json:"path"`
}

// Project отображает скрутуру проектов
type Project struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
}

// Allower
type Allower struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
}

// Config отображает структуру файла конфигураций, откуда парсятся данные для
// доступа к gitlab
type Config struct {
	GitlabURLSource    string `json:"gitlabURLSource"`
	PrivateTokenSource string `json:"privateTokenSource"`
	GitlabURLDest      string `json:"gitlabURLDest"`
	PrivateTokenDest   string `json:"privateTokenDest"`
}

const (
	xxxArea            = "mock-sync"
	exportCheckPeriod  = 5 * time.Second
	whiteListGroupPath = "mock"
	tmpDir             = "./cloneProjects"
	reservationAddress = "https://xxx.xxx.xx.xx"  // Адрес резервации для использования black list
	destAddress        = "https://git.ixample.ru" // Конечный адрес для использования black list
)

// Black листа два. 1. На синхронизацию с резервацией 2. На синхронизацию резеровации с xxxxx
// От xxxxx полностью изолировать группы проектов искра, art отдел, группу проектов xxxx и группу проектов xxxx и группу проектов xxxx
func main() {
	// Установим счетчик времени
	currentTime := time.Now()
	// Настроим рабочую директорию
	if err := setUpWorkspace(); err != nil {
		fmt.Printf("[ERROR] Failed to set up working space: %v\n", err)
		return
	}
	// Создадим файл для сохранения логов поврежденных проектов в репозитории (которые невозможно загрузить)
	corruptedFile, err := os.OpenFile("currupted-projects.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open log file: %v\n", err)
		return
	}
	defer corruptedFile.Close()
	// Создадим файл для общего логирования
	generalFile, err := os.OpenFile("general.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open general log file: %v\n", err)
		return
	}
	defer generalFile.Close()
	// Устанавливаем логгеры для различных типов логов
	corruptedLogger := log.New(corruptedFile, "CORRUPTED: ", log.Ldate|log.Ltime|log.Lshortfile)
	generalLogger := log.New(generalFile, "GENERAL: ", log.Ldate|log.Ltime|log.Lshortfile)
	// Отделим логи от предыдущего вызова программы
	generalLogger.Printf("------------ %s ------------\n", currentTime)
	corruptedLogger.Printf("------------ %s ------------\n", currentTime)
	fmt.Printf("[START] Gitlab importer start now: %s\n", currentTime)
	generalLogger.Printf("[START] Gitlab importer start now: %s\n", currentTime)
	// Чтение содержимого файла конфигурации
	configFile := "creds.json"
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("[ERROR] Failed to read config file: %v\n", err)
		generalLogger.Printf("[ERROR] Failed to read config file: %v\n", err)
		os.Exit(1)
	}
	// Создание структуры для хранения конфигурации
	var config Config
	// Декодирование JSON в структуру
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("[ERROR] Failed to parse config file: %v\n", err)
		generalLogger.Printf("[ERROR] Failed to parse config file: %v\n", err)
		os.Exit(1)
	}
	// Получим корневые группы
	rootGroups, err := getRootGroups(generalLogger, config.GitlabURLSource, config.PrivateTokenSource)
	if err != nil {
		fmt.Println("[ERROR] Error fetching root groups with parent_(id=0):", err)
		generalLogger.Println("[ERROR] Error fetching root groups with parent_(id=0):", err)
		os.Exit(1)
	}
	// Создадим группу xxxxx-sync, в которую будут записываться проекты и группы на удаленном Gitlab-destination
	// если группа существует, то просто получим её ID
	xxxAreaGroupID := createGroup(generalLogger, config.GitlabURLDest, config.PrivateTokenDest, Group{Name: xxxArea, Path: xxxArea, FullPath: xxxArea}, 0, true)
	// blackList :=
	// Пройдемся по всем КОРНЕВЫМ группам в родном Gitlab-source
	for _, group := range rootGroups {
		// Выставим ограничение на загрузку только xxx-dep и xxx в резервацию
		if !(config.GitlabURLDest == destAddress) {
			if !(group.FullPath == "xxxxx") && !(group.FullPath == "xxxxx-dep") {
				continue
			}
		}

		// Следующая проверка позволяет загрузить в xxxxx Gitlab исключительно проекты из группы xxx-sync,
		// но также позволяет загружать проекты из всех групп, если это не xxxxx (т.е. если это резервация)
		if config.GitlabURLDest == destAddress {
			if !strings.HasPrefix(group.FullPath, xxxArea) {
				continue
			}
		}

		// if group.FullPath == "xxxxx" {

		importProjectClone(config, group, generalLogger, corruptedLogger, xxxAreaGroupID)

		// }
		// Вызовем объединенную функцию импортирования проектов
		// importProcessArchive(group, config, xxxAreaGroupID)
	}
	// Удаляем бейдж private c корневой директории xxxxx-sync в резервации
	_, xxxArexxxAreaGroupBadgeID := getBadge(config.GitlabURLDest, config.PrivateTokenDest, xxxAreaGroupID)
	if xxxArexxxAreaGroupBadgeID != 0 {
		err = removeBadge(config.GitlabURLDest, config.PrivateTokenDest, xxxAreaGroupID, xxxArexxxAreaGroupBadgeID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to remove badge for group %s: %v\n", xxxArea, err)
			generalLogger.Printf("[ERROR] Failed to remove badge for group %s: %v\n", xxxArea, err)
			os.Exit(1)
		}
	}
	// Выводим время выполнения программы и завершаем её
	endTime := time.Since(currentTime)
	fmt.Printf("[END] Program complete at: %v\n", endTime)
	generalLogger.Printf("[END] Program complete at: %v\n", endTime)
	os.Exit(0)
}

// Проверка на "экспортирован ли проект?" и возвращает статус экспорта
func isExportFinished(url, token string, projectID int) (bool, string) {
	fmt.Println("[DEBUG] isExportFinished-> Check export status id: ", projectID)
	fmt.Println("[DEBUG] isExportFinished-> Check export status id: ", projectID)
	// СОздаем запрос
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/projects/%d/export", url, projectID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполняем и сохраняем ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error checking export status:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	// Сохраняем тело ответа в структуру
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("[ERROR] Error decoding response:", err)
		os.Exit(1)
	}
	// Сохраняем статус экспорта (none, started, finished, failed)
	status, ok := result["export_status"].(string)

	fmt.Println("[DEBUG] isExportFinished<- export status is: ", status)
	return ok && status == "finished", status
}

// Загрузка файла из на локальную машину
func downloadProject(url, token string, projectID int, projectName string) error {
	fmt.Println("[DEBUG] downloadProject-> Start download project to local machine. Project: ", projectName)
	// Создадим запрос на загрузку файла на локальную машину
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/projects/%d/export/download", url, projectID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполним запрос и сохраним ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error downloading project:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return errors.New("[WARNING] Network is buisy, retry automatic download")
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[ERROR] Failed to download project. Status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	// Создадим файл для записи полученных данных с Gitlab-source
	file, err := os.Create(fmt.Sprintf("%s.tar.gz", projectName))
	if err != nil {
		fmt.Println("[ERROR] Error creating file:", err)
		os.Exit(1)
	}
	defer file.Close()
	// Скопируем полученные из сети данные в созданный файл
	if _, err := io.Copy(file, resp.Body); err != nil {
		fmt.Println("[ERROR] Error saving file:", err)
		os.Exit(1)
	}
	fmt.Println("[SUCCESS] downloadProject<- Download complete: ", projectName)
	return nil
}

// Экспортируем проект
func exportProject(url, token string, projectID int) {
	fmt.Println("[DEBUG] exportProject-> Exporting project ID: ", projectID)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v4/projects/%d/export", url, projectID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error exporting project:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Println("[SUCCESS] exportProject<- Project exported: ", projectID)
}

// Импортирование проекта на Gitlab-destination
func importProject(url, token string, projectName, groupPath string) {
	fmt.Printf("[DEBUG] importProject-> Importing project: %s\n                 Path in group: %s\n", projectName, groupPath)
	// ЧИтаем файл, который мы хотим импортировать
	file, err := os.Open(fmt.Sprintf("%s.tar.gz", projectName))
	if err != nil {
		fmt.Println("[ERROR] Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()
	// Создадим канал для записи импортируемого файла
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	// СОздадим анонимную горутину для записи файла по частям (ибо на выгрузку файла целиком может не хватить памяти оперативной)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
		if err != nil {
			fmt.Println("[ERROR] Error creating form file:", err)
			os.Exit(1)
		}
		if _, err := io.Copy(part, file); err != nil {
			fmt.Println("[ERROR] Error copying file:", err)
			os.Exit(1)
		}
		writer.Close()
	}()
	// СОздадим запрос на импорт файла записанного в pr
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v4/projects/import?path=%s&namespace=%s&overwrite=true", url, projectName, groupPath), pr)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполним запрос и сохраним ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error importing project:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	// Прочитаем тело ответа и выведем его
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("[ERROR] Failed to import project. Status: %d, Response: %s\n", resp.StatusCode, string(respBody))
	} else {
		fmt.Println("[SUCCESS] importProject<- Project imported successfully: ", projectName)
	}
}

// Получим все проекты в конкретной группе
func getProjectsFromGroup(generalLogger *log.Logger, url, token string, groupID int) []Project {
	fmt.Println("[DEBUG] getProjectsFromGroup-> Getting projects from group ID: ", groupID)
	generalLogger.Println("[DEBUG] getProjectsFromGroup-> Getting projects from group ID: ", groupID)
	var projects []Project
	for i := 1; i <= 10; i++ {
		// Конструируем запрос
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/groups/%d/projects?per_page=100&page=%d", url, groupID, i), nil)
		if err != nil {
			fmt.Println("[ERROR] Error creating request:", err)
			generalLogger.Println("[ERROR] Error creating request:", err)
			os.Exit(1)
		}
		req.Header.Set("PRIVATE-TOKEN", token)
		// Настройка транспорта с отключенной проверкой SSL-сертификатов
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		// Создание HTTP-клиента с настраиваемым транспортом
		client := &http.Client{Transport: tr}
		// Выполним запрос на получение проектов и сохраним ответ
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("[ERROR] Error getting projects:", err)
			generalLogger.Println("[ERROR] Error getting projects:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var projectsPerPage []Project
		// Парсим ответ в структуру
		if err := json.NewDecoder(resp.Body).Decode(&projectsPerPage); err != nil {
			fmt.Println("[ERROR] Error decoding response:", err)
			generalLogger.Println("[ERROR] Error decoding response:", err)
			os.Exit(1)
		}
		if len(projectsPerPage) == 0 {
			break
		}
		projects = append(projects, projectsPerPage...)

	}
	fmt.Printf("[SUCCESS] getProjectsFromGroup<- Project was got:\n            %v\n", projects)
	generalLogger.Printf("[SUCCESS] getProjectsFromGroup<- Project was got:\n            %v\n", projects)
	return projects
}

// Парсим дерево подгрупп и выполняем аналогичные действия, действиям с root группами
func parseSubgroupTree(config Config, generalLogger, corruptedLogger *log.Logger, IDSrc, parentIDDst int) {
	fmt.Println("[DEBUG]-> Subdirectory operations start")
	// Получаем список подгрупп по ID
	subgroups, err := getSubgroupsInGroup(generalLogger, fmt.Sprintf("%s/api/v4", config.GitlabURLSource), config.PrivateTokenSource, IDSrc)
	if err != nil {
		fmt.Println("[ERROR] Error fetching subgroups:", err)
		generalLogger.Println("[ERROR] Error fetching subgroups:", err)
		os.Exit(1)
	}
	// Пройдемся по каждой подгруппе
	for _, subgroup := range subgroups {
		importProjectClone(config, subgroup, generalLogger, corruptedLogger, parentIDDst)
		// Вызовем объединенную функцию импортирования проектов
		// importProcessArchive(subgroup, config, parentIDDst)
	}
	fmt.Println("[DEBUG]<- Subdirectory operations end")
	generalLogger.Println("[DEBUG]<- Subdirectory operations end")
}

// Получим список корневых групп
func getRootGroups(generalLogger *log.Logger, url, PrivateTokenSource string) ([]Group, error) {
	fmt.Println("[DEBUG] getRootGroups-> Getting root groups list from Gitlab-source")
	generalLogger.Println("[DEBUG] getRootGroups-> Getting root groups list from Gitlab-source")
	var allGroups []Group
	var rootGroups []Group
	// Достроим URL
	groupsURL := fmt.Sprintf("%s/api/v4/groups?per_page=100&page=1", url)
	// Создадим запрос
	req, err := http.NewRequest("GET", groupsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to create request: %v", err)
	}
	req.Header.Set("Private-Token", PrivateTokenSource)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполним запрос
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	// respBody, _ := io.ReadAll(resp.Body)
	// fmt.Printf("%s\n", respBody)
	//  Проверим статус код ответа (должен быть 200)
	if resp.StatusCode != http.StatusOK {
		// return nil, fmt.Errorf("[ERROR] GitLab API request failed with status code %d\n        reason: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("[ERROR] GitLab API request failed with status code %d\n        reason: ", resp.StatusCode)

	}

	// Декодируем ответ в структуру
	err = json.NewDecoder(resp.Body).Decode(&allGroups)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Failed to decode JSON response: %v", err)
	}

	// Сохраним только группы с id=0 (ибо нам нужны сейчас только корневые)
	for _, group := range allGroups {
		if group.ParentID == 0 {
			rootGroups = append(rootGroups, group)
		}
	}
	fmt.Printf("[SUCCESS] getRootGroups<- Root groups got:\n            %v\n", rootGroups)
	generalLogger.Printf("[SUCCESS] getRootGroups<- Root groups got:\n            %v\n", rootGroups)
	return rootGroups, nil
}

// Получаем список подгрупп
func getSubgroupsInGroup(generalLogger *log.Logger, url, PrivateTokenSource string, parentID int) ([]Group, error) {
	fmt.Println("[DEBUG] getSubgroupsInGroup-> Getting subgpoups into group ID=", parentID)
	generalLogger.Println("[DEBUG] getSubgroupsInGroup-> Getting subgpoups into group ID=", parentID)
	var subgroups []Group
	// Конструируем корректный URL
	subgroupsURL := fmt.Sprintf("%s/groups/%d/subgroups", url, parentID)
	// Создаем запрос
	req, err := http.NewRequest("GET", subgroupsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Private-Token", PrivateTokenSource)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполним запрос и соххраним ответ
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()
	// Проверим код http, должен быть 200
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[ERROR] GitLab API request failed with status code %d", resp.StatusCode)
	}
	// Сохраним тело ответа в структуре
	err = json.NewDecoder(resp.Body).Decode(&subgroups)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}
	fmt.Printf("[SUCCESS] getSubgroupsInGroup<- Subgroups got: %v\n", subgroups)
	generalLogger.Printf("[SUCCESS] getSubgroupsInGroup<- Subgroups got: %v\n", subgroups)
	return subgroups, nil
}

// Создание группы
func createGroup(generalLogger *log.Logger, url, token string, group Group, parentID int, parentIsRoot bool) int {
	fmt.Println("[DEBUG] createGroup-> Creating group in Gitlab-destination: ", group.Name)
	generalLogger.Println("[DEBUG] createGroup-> Creating group in Gitlab-destination: ", group.Name)
	// Проверим, существует ли такая группа, если да -- вернем её ID и завершим функцию
	existingGroup := getGroup(generalLogger, url, token, group.FullPath, parentID, parentIsRoot)
	if existingGroup != nil {
		return existingGroup.ID
	}
	// Если группы нет -- продолжим создание
	// присвоим имя и путь из аргумента функции
	data := map[string]interface{}{
		"name": group.Name, // Имя группы для GUI
		"path": group.Path, // Путь группы для URL
	}
	if parentID != 0 {
		data["parent_id"] = parentID
	}
	fmt.Printf("[DEBUG] Creating group: %+v\n", data)
	generalLogger.Printf("[DEBUG] Creating group: %+v\n", data)
	// Замаршалим данные для передачи в тело запроса
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("[ERROR] Error marshaling request data:", err)
		generalLogger.Println("[ERROR] Error marshaling request data:", err)
		return parentID
	}
	// Построим запрос
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v4/groups", url), bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		generalLogger.Println("[ERROR] Error creating request:", err)
		return parentID
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполним запрос на создание группы и сохраним ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error creating group:", err)
		generalLogger.Println("[ERROR] Error creating group:", err)
		return parentID
	}
	defer resp.Body.Close()
	// Проверим код ответа (должен быть 201)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[ERROR] Failed to create group. Status: %d, Response: %s\n", resp.StatusCode, string(body))
		generalLogger.Printf("[ERROR] Failed to create group. Status: %d, Response: %s\n", resp.StatusCode, string(body))
		return parentID
	}

	var createdGroup Group
	if err := json.NewDecoder(resp.Body).Decode(&createdGroup); err != nil {
		fmt.Println("[ERROR] Error decoding response:", err)
		generalLogger.Println("[ERROR] Error decoding response:", err)
		return parentID
	}

	fmt.Println("[SUCCESS] createGroup<- Group created: ", group.Name)
	generalLogger.Println("[SUCCESS] createGroup<- Group created: ", group.Name)
	return createdGroup.ID
}

// Получаем данные о существующей группы, или возвращаем nil, если таковой не существует
func getGroup(generalLogger *log.Logger, url, token, fullPath string, parentID int, parentIsRoot bool) *Group {
	fmt.Println("[DEBUG] getGroup-> Getting existing group info")
	generalLogger.Println("[DEBUG] getGroup-> Getting existing group info")
	var err error
	var req *http.Request
	client := &http.Client{}
	// Конструируем разные запросы в зависимости от родительской группы (находится ли в корне или группе?)
	if parentIsRoot {
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/api/v4/groups/%s", url, fullPath), nil)
	} else {
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/api/v4/groups/%d/subgroups", url, parentID), nil)
	}

	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		generalLogger.Println("[ERROR] Error creating request:", err)
		return nil
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client = &http.Client{Transport: tr}
	// Выполним запрос и получим ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error getting group:", err)
		generalLogger.Println("[ERROR] Error getting group:", err)
		return nil
	}
	defer resp.Body.Close()
	// Получим ответ -- nil будет означать что группы нет, можно завершать проверку
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	var group Group
	var groups []Group
	// Парсим ответ в зависимости от типа запроса: к root group или к subgroup
	if parentIsRoot {
		if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
			fmt.Println("[ERROR] Error decoding response:", err)
			generalLogger.Println("[ERROR] Error decoding response:", err)
			return nil
		}
		// Возвращаем ответ в какой он и пришел (декодируя в структуру одиночной группы)
		return &group
		// Если же запрос был НЕ к корню
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
			fmt.Println("[ERROR] Error decoding response:", err)
			generalLogger.Println("[ERROR] Error decoding response:", err)
			return nil
		}
		// Необходимо распарсить ответ и вычленить искомую группу. Таким образом
		// мы сохраним ID, Parent ID и path, уже существующей группы (эти параметры необхогдимы для создания подгрупп
		// и выгрузки проектов в группу)
		for _, group := range groups {
			subgroupNameSplitter := strings.Split(fullPath, "/")
			subgroupName := subgroupNameSplitter[len(subgroupNameSplitter)-1]
			if group.Path == subgroupName {
				return &group
			}
		}
	}
	fmt.Printf("[SUCCESS] getGroup<- Group info was got:\n            %v\n", &group)
	generalLogger.Printf("[SUCCESS] getGroup<- Group info was got:\n            %v\n", &group)
	return nil
}

// Функция занимается полным процессом импорта проекта
func importProcessArchive(group Group, config Config, corruptedLogger *log.Logger, generalLogger *log.Logger, parentGroupID int) {
	fmt.Printf("[DEBUG] importProcessArchive-> Start importing group: %s; Path: %s\n", group.Name, group.FullPath)
	generalLogger.Printf("[DEBUG] importProcessArchive-> Start importing group: %s; Path: %s\n", group.Name, group.FullPath)
	// Заменим полный путь корневой группы из Gitlab-source, добавив директорию xxx-sync для импорта в Gitlab-destination
	group.FullPath = fmt.Sprintf("%s/%s", xxxArea, group.FullPath)
	// Создадим группу в корне
	parentID := createGroup(generalLogger, config.GitlabURLDest, config.PrivateTokenDest, group, parentGroupID, false)
	// Получим все проекты в группе из Gitlab-source
	fmt.Println("[DEBUG] Group name to getting projects: ", group.Name)
	generalLogger.Println("[DEBUG] Group name to getting projects: ", group.Name)
	projects := getProjectsFromGroup(generalLogger, config.GitlabURLSource, config.PrivateTokenSource, group.ID)
	// Пройдемся по всем полученым проектам
ProjectLoop:
	for _, project := range projects {
		// Экспортируем проект (да, без этого мы не сможем его загрузить на локальную машину)
		fmt.Println("[DEBUG] Project name to export: ", project.Name)
		generalLogger.Println("[DEBUG] Project name to export: ", project.Name)
		exportProject(config.GitlabURLSource, config.PrivateTokenSource, project.ID)
		// Проверим, экспортировался проект или нет, если нет, то подождем 5 сек
		// если проект по каким-то причинам не может быть экспортирован (покаррапчен, ибо в таком случае
		// и clone работать не будет), то преррываем этот проект перейдя к следующему
		// Количество попыток для ошибочного вызова со статусом none
		try := 0
		for {
			finished, status := isExportFinished(config.GitlabURLSource, config.PrivateTokenSource, project.ID)
			if finished {
				break
			}
			if status == "none" && try > 15 {
				fmt.Printf("[ERROR] Export not can be finished, observe project: %s ID: %d\n", project.Name, project.ID)
				// Пишем логи
				corruptedLogger.Printf("Project currupted: %d;%s\n", project.ID, project.Name)
				continue ProjectLoop
			}
			time.Sleep(exportCheckPeriod)
			try++
		}
		// Далее будет загрузка на локальный пк проекта. Цикл необходим для корректной загрузки во избежании
		// ошибки http 429 (слишком частные запросы к ресурсу)
		for {
			// Загружаем проект на локальную машину и проверяем ошибку http 429
			warn := downloadProject(config.GitlabURLSource, config.PrivateTokenSource, project.ID, project.Name)
			if warn == nil {
				// Если ошибки нет, завершаем загрузку. Если есть -- ждем 5 сек, и после повторяем загрузку
				break
			}
			fmt.Println(warn)
			time.Sleep(exportCheckPeriod)
		}
		// Импортируем проект (выгружаем его) на Gitlab-destination
		importProject(config.GitlabURLDest, config.PrivateTokenDest, project.Name, group.FullPath)
		// На этом этапе с корневыми проектами и группами покончено
		// необходимость разделять на корневые проекты (точнее проекты находящиеся в группах, лежащих в корне)
		// появляется из-за разности в запросах на группы в корне (группы) между запросами на группы в группах (подгруппы)
	}
	// Создаем дерево подгрупп и импортируем проекты из подгрупп
	parseSubgroupTree(config, generalLogger, corruptedLogger, group.ID, parentID)
	fmt.Printf("[SUCCESS] importProcessArchive<- End of importing group: %s; Path: %s\n", group.Name, group.FullPath)
	generalLogger.Printf("[SUCCESS] importProcessArchive<- End of importing group: %s; Path: %s\n", group.Name, group.FullPath)
}

// Функция для удаления группы
func deleteGitLabGroup(GitlabURLDest, PrivateTokenDest string, groupID int) error {
	fmt.Println("[DEBUG] deleteGitLabGroup-> Removing group ID: ", groupID)
	url := fmt.Sprintf("%s/api/v4/groups/%d", GitlabURLDest, groupID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("[ERROR] Error with creating request DELETE: %w", err)
	}

	req.Header.Set("Private-Token", PrivateTokenDest)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[ERROR] Error with creating request DELETE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("[ERROR] Error with creating request DELETE: Status code: %d. Body: %s", resp.StatusCode, string(body))
	}
	fmt.Println("[SUCCESS] deleteGitLabGroup<- Group removed group ID: ", groupID)
	return nil
}

// cloneRepo клонирует репозиторий с исходного Gitlab
func cloneRepo(generalLogger, corruptedLogger *log.Logger, repoURL, destDir string) error {
	cmd := exec.Command("git", "clone", "--mirror", repoURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] Failed to clone repository: %v\n", err)
		generalLogger.Printf("[ERROR] Failed to clone repository: %v\n", err)
		corruptedLogger.Printf("Cloning currupted, URL: %s\n", repoURL)
		return err
	}

	// Стянуть все LFS объекты
	lfsCmd := exec.Command("git", "-C", destDir, "lfs", "fetch", "--all")
	lfsCmd.Stdout = os.Stdout
	lfsCmd.Stderr = os.Stderr
	if err := lfsCmd.Run(); err != nil {
		// Временно поставил nil, но нужно что-то с этим придумать
		corruptedLogger.Println("LFS currupted, URL: ", repoURL)
		return err
	}

	return nil
}

// pushRepo пушит репозиторий на удалённый Gitlab
func pushRepo(generalLogger *log.Logger, repoDir, newRepoURL string) error {
	// Создание новой переменной окружения только для текущего процесса
	// env := os.Environ()
	// env = append(env, remoteSSHJump)
	// Пушим все LFS объекты
	// lfsCmd := exec.Command("git", "-c", "http.sslVerify=false", "-C", repoDir, "lfs", "push", "--all", newRepoURL)
	lfsCmd := exec.Command("git", "-C", repoDir, "lfs", "push", "--all", newRepoURL)
	// lfsCmd := exec.Command("bash", "-c", fmt.Sprintf("GIT_SSH_COMMAND='%s' git -C %s lfs push --all %s", remoteSSHJump, repoDir, newRepoURL))
	// lfsCmd.Env = env
	lfsCmd.Stdout = os.Stdout
	lfsCmd.Stderr = os.Stderr
	if err := lfsCmd.Run(); err != nil {
		fmt.Printf("[WARNING] Failed to push lfs: %v\n", err)
		generalLogger.Printf("[WARNING] Failed to push lfs: %v\n", err)
		// Временная мера
		return nil
	}

	// Получаем список всех веток
	branchesCmd := exec.Command("git", "-C", repoDir, "for-each-ref", "--format=%(refname)", "refs/heads/")
	branchesOutput, err := branchesCmd.Output()
	if err != nil {
		fmt.Printf("[ERROR] Failed to get all branches: %v\n", err)
		generalLogger.Printf("[ERROR] Failed to get all branches: %v\n", err)
		return err
	}
	branches := strings.Fields(string(branchesOutput))

	// Получаем список всех тегов
	tagsCmd := exec.Command("git", "-C", repoDir, "for-each-ref", "--format=%(refname)", "refs/tags/")
	tagsOutput, err := tagsCmd.Output()
	if err != nil {
		fmt.Printf("[ERROR] Failed get all tags: %v\n", err)
		generalLogger.Printf("[ERROR] Failed get all tags: %v\n", err)
		return err
	}
	tags := strings.Fields(string(tagsOutput))

	// Пушим все ветки
	for _, branch := range branches {
		cmd := exec.Command("git", "-C", repoDir, "push", newRepoURL, branch, "--force")
		// cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("[ERROR] Failed to push branch: %s; error: %v\n", branch, err)
			generalLogger.Printf("[ERROR] Failed to push branch: %s; error: %v\n", branch, err)
			// Временно
			return nil
		}
	}

	// Пушим все теги
	for _, tag := range tags {
		cmd := exec.Command("git", "-C", repoDir, "push", newRepoURL, tag, "--force")
		// cmd := exec.Command("sh", "-c", fmt.Sprintf("GIT_SSH_COMMAND='%s' git -C %s push %s %s", remoteSSHJump, repoDir, newRepoURL, tag))
		// cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("[ERROR] Failed to push tag: %s; error: %v\n", tag, err)
			generalLogger.Printf("[ERROR] Failed to push tag: %s; error: %v\n", tag, err)
			return err
		}
	}

	return nil
	// // Пушим репозиторий
	// cmd := exec.Command("git", "-C", repoDir, "push", "--mirror", newRepoURL)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// if err := cmd.Run(); err != nil {
	// 	return err
	// }

	// return nil
}

// cleanUp полностью очищает ЛОКАЛЬНЫЙ репозиторий
func cleanUp(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Printf("[ERROR] Failed: %v\n", err)
		return err
	}
	return os.Mkdir(tmpDir, 0755)
}

// importProjectClone импортирует проекты путём клонирования/пуша
func importProjectClone(config Config, group Group, generalLogger, corruptedLogger *log.Logger, parentGroupID int) {
	fmt.Printf("[DEBUG] importProjectClone-> Start importing group: %s; Path: %s\n", group.Name, group.FullPath)
	// Устанавливаем удаленный порт в зависимости от получателя (у xxx это 22, а резервация -- 2222)
	destSSHPortPostfix := "2222"
	if config.GitlabURLDest == destAddress {
		destSSHPortPostfix = "22"
	}
	// Фильтруем группы и подгруппы, которые хотим переносить на Gtilab destination
	badge, _ := getBadge(config.GitlabURLSource, config.PrivateTokenSource, group.ID)
	// if config.GitlabURLDest == reservationAddress && badge == "private" {
	// 	return
	// } else if config.GitlabURLDest == destAddress && badge != "xxx" {
	// 	return
	// }
	if config.GitlabURLDest == destAddress && badge == "private" {
		return
	}
	// Создадим группу на удаленном Gitlab, если это не xxx-sync, воизбежании рекурсивного создани директории xxx-sync
	var parentID int
	if group.Path == xxxArea {
		parentID = parentGroupID
	} else {
		parentID = createGroup(generalLogger, config.GitlabURLDest, config.PrivateTokenDest, group, parentGroupID, false)
	}
	// Применим бэйдж из исходного Gitlab на удаленный
	if badge != "" && config.GitlabURLDest != destAddress {
		// проверим установлен ли уже бейдж
		existingBadge, _ := getBadge(config.GitlabURLDest, config.PrivateTokenDest, parentID)
		// И если бейдж не установлен, установим
		if existingBadge == "" {
			setBadge(config.GitlabURLDest, config.PrivateTokenDest, badge, parentID)
		}
	}
	// Получим все проекты в группе из Gitlab-source
	fmt.Println("[DEBUG] Group name to getting projects: ", group.Name)
	generalLogger.Println("[DEBUG] Group name to getting projects: ", group.Name)
	projects := getProjectsFromGroup(generalLogger, config.GitlabURLSource, config.PrivateTokenSource, group.ID)
	// Пройдемся по всем полученым проектам
	for _, project := range projects {
		// Заменим все пробьелы дефисом в имени проекта
		project.Name = strings.ReplaceAll(project.Name, " ", "-")
		// Привдем к раочему виду строки для адресов репозиториев в соответствии с ssh форматом
		modifiedGitlabURLSource := strings.TrimPrefix(config.GitlabURLSource, "https://")
		modifiedGitlabURLDest := strings.TrimPrefix(config.GitlabURLDest, "https://")
		// modifiedGitlabURLDest = strings.TrimPrefix(config.GitlabURLDest, "http://")
		sourceRepoURL := fmt.Sprintf("ssh://git@%s:2222/%s/%s.git", modifiedGitlabURLSource, group.FullPath, project.Name)
		groupDest := xxxArea + "/" + group.FullPath
		if config.GitlabURLDest == destAddress {
			groupDest = group.FullPath
		}
		destRepoURL := fmt.Sprintf("ssh://git@%s:%s/%s/%s.git", modifiedGitlabURLDest, destSSHPortPostfix, groupDest, project.Name)
		repoName := filepath.Base(sourceRepoURL)
		repoName = repoName[:len(repoName)-len(filepath.Ext(repoName))]
		//  Зададим имя репозитория
		tempRepoDir := filepath.Join(tmpDir, repoName+".git")
		// Скопируем репозиторий с Gitlab-source
		fmt.Printf("[DEBUG] Cloning repository from %s...\n", sourceRepoURL)
		generalLogger.Printf("[DEBUG] Cloning repository from %s...\n", sourceRepoURL)
		if err := cloneRepo(generalLogger, corruptedLogger, sourceRepoURL, tempRepoDir); err != nil {
			fmt.Printf("[ERROR] Failed to clone repository: %v\n", err)
			generalLogger.Printf("[ERROR] Failed to clone repository: %v\n", err)
			continue
		}
		// Запушим склонированный репозиторий на удаленный Gitlab-destination
		fmt.Printf("[DEBUG] Pushing repository to %s...\n", destRepoURL)
		generalLogger.Printf("[DEBUG] Pushing repository to %s...\n", destRepoURL)
		if err := pushRepo(generalLogger, tempRepoDir, destRepoURL); err != nil {
			fmt.Printf("[ERROR] Failed to push repository: %v\n", err)
			generalLogger.Printf("[ERROR] Failed to push repository: %v\n", err)
			os.Exit(1)
		}
		// Очистим директорию с локальным репозиторием
		fmt.Printf("[DEBUG] Cleaning up temporary files...\n")
		generalLogger.Printf("[DEBUG] Cleaning up temporary files...\n")
		if err := cleanUp(tmpDir); err != nil {
			fmt.Printf("[ERROR] Failed to clean up: %v\n", err)
			generalLogger.Printf("[ERROR] Failed to clean up: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[SUCCESS] Repository transfer complete!")
		generalLogger.Println("[SUCCESS] Repository transfer complete!")
	}
	// А Это мы выставляем разрешение на force push
	destinationProjects := getProjectsFromGroup(generalLogger, config.GitlabURLDest, config.PrivateTokenDest, parentID)
	for _, destProject := range destinationProjects {
		defaultBranchName, err := getProjectDefaultBranch(config.GitlabURLDest, config.PrivateTokenDest, destProject.ID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to getting default project branch name: %v\n", err)
			generalLogger.Printf("[ERROR] Failed to getting default project branch name: %v\n", err)
		}
		err = allowForcePush(config.GitlabURLDest, defaultBranchName, config.PrivateTokenDest, destProject.ID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to remove force push option: %v\n", err)
			generalLogger.Printf("[ERROR] Failed to remove force push option: %v\n", err)
		}
	}
	//
	// Создаем дерево подгрупп и импортируем проекты из подгрупп
	parseSubgroupTree(config, generalLogger, corruptedLogger, group.ID, parentID)
	fmt.Printf("[SUCCESS] importProjectClone<- End of importing group: %s; Path: %s\n", group.Name, group.FullPath)
	generalLogger.Printf("[SUCCESS] importProjectClone<- End of importing group: %s; Path: %s\n", group.Name, group.FullPath)
}

// setUpWorkspace применяет директорию с исполняемым файлом программы как рабочую директорию
func setUpWorkspace() error {
	// Получаем путь к исполняемому файлу
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("[ERROR] Failed: %v\n", err)
	}
	// Получаем директорию исполняемого файла
	exeDir := filepath.Dir(exePath)
	// Устанавливаем эту директорию как текущую рабочую директорию
	err = os.Chdir(exeDir)
	if err != nil {
		fmt.Printf("[ERROR] Failed: %v\n", err)
	}
	// Удалим директорию с (о вдруг) старыми проектами
	err = os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Printf("[ERROR] Failed: %v\n", err)
	}
	// Создаем временную директорию для временного хранения склонированных репозиториев
	err = os.Mkdir(tmpDir, 0755)
	if err != nil {
		fmt.Printf("[ERROR] Failed: %v\n", err)
		return err
	}
	return nil
}

// getBadge получает badge указанной группы
func getBadge(url, token string, groupID int) (string, int) {
	// Создаем запрос
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/groups/%d/badges", url, groupID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		os.Exit(1)
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// Создаем HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполняем и сохраняем ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error checking export status:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result []Group
	// Сохраняем тело ответа в структуру
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("[ERROR] Error decoding response:", err)
		os.Exit(1)
	}
	if len(result) != 0 {
		return result[0].Name, result[0].ID
	}
	return "", 0
}

// setBadge устанавливает badge на группу
func setBadge(url, token, newBadgeName string, groupID int) {
	fmt.Println("[DEBUG] setBadge-> start")
	// Данные для бейджа
	badgeData := BadgeData{
		Name:     newBadgeName,
		LinkURL:  "https://example.com",
		ImageURL: "https://example.com/badge.svg",
	}
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// Создаем HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}

	body, err := json.Marshal(badgeData)
	if err != nil {
		fmt.Printf("[ERROR] Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	// Выполняем и сохраняем ответ
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v4/groups/%d/badges", url, groupID), bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("[ERROR] Failed to create request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Failed to send request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("[SUCCESS] setBadge<- badge created successfully.")
	} else {
		fmt.Printf("[ERROR] Failed to create badge: %d - %s\n", resp.StatusCode, resp.Status)
	}
}

// removeBadge удалит бейдж с группы
func removeBadge(url, token string, groupID, badgeID int) error {
	fmt.Println("[DEBUG] removeBadge-> start, group id:", groupID)
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v4/groups/%d/badges/%d", url, groupID, badgeID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// Создаем HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error making request:", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("[ERROR] Error reading response body:", err)
		return err
	}

	if resp.StatusCode == http.StatusNoContent {
		fmt.Println("[SUCCESS] removeBadge<- Badge deleted successfully")
	} else {
		fmt.Printf("[ERROR] Failed to delete badge. Status code: %d, Response: %s\n", resp.StatusCode, body)
		return err
	}

	return nil
}

// getProjectDefaultBranch получает имя ветки по умолчанию
func getProjectDefaultBranch(url, token string, projectID int) (string, error) {
	fmt.Println("[DEBUG] getProjectDefaultBranch-> getting default branch id: ", projectID)
	// СОздаем запрос
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v4/projects/%d", url, projectID), nil)
	if err != nil {
		fmt.Println("[ERROR] Error creating request:", err)
		return "", err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	// Выполняем и сохраняем ответ
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[ERROR] Error checking export status:", err)
		return "", err
	}
	defer resp.Body.Close()

	var projectInfo Project
	// Парсим ответ в структуру
	if err := json.NewDecoder(resp.Body).Decode(&projectInfo); err != nil {
		fmt.Println("[ERROR] Error decoding response:", err)
		// generalLogger.Println("[ERROR] Error decoding response:", err)
		return "", err
	}
	fmt.Println("[SUCCESS] getProjectDefaultBranch<- default branch got: ", projectInfo.DefaultBranch)
	return projectInfo.DefaultBranch, nil
}

// allowForcePush разврешает force push
func allowForcePush(url, branchName, token string, projectID int) error {
	fmt.Println("[DEBUG] allowForcePush-> removing force push for: ", projectID, branchName)
	constructUrl := fmt.Sprintf("%s/api/v4/projects/%d/protected_branches/%s", url, projectID, branchName)

	req, err := http.NewRequest("DELETE", constructUrl, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	// Настройка транспорта с отключенной проверкой SSL-сертификатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Создание HTTP-клиента с настраиваемым транспортом
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("[ERROR] Error with creating request DELETE: Status code: %d. Body: %s", resp.StatusCode, string(body))
	}
	fmt.Println("[SUCCESS] allowForcePush-> removing force push for: ", projectID, branchName)
	return nil
}

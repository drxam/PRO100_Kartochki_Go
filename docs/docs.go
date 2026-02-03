// Package docs МозгоЁмка API.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "REST API для приложения карточек МозгоЁмка",
        "title": "МозгоЁмка API",
        "version": "1.0"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/auth/register": {
            "post": {
                "tags": ["auth"],
                "summary": "Регистрация",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/RegisterRequest"}}],
                "responses": {"201": {"description": "Created"}, "400": {"description": "Bad Request"}}
            }
        },
        "/auth/login": {
            "post": {
                "tags": ["auth"],
                "summary": "Вход",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/LoginRequest"}}],
                "responses": {"200": {"description": "OK"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/auth/refresh": {
            "post": {
                "tags": ["auth"],
                "summary": "Обновление токенов",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/RefreshRequest"}}],
                "responses": {"200": {"description": "OK"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/auth/logout": {
            "post": {
                "security": [{"BearerAuth": []}],
                "tags": ["auth"],
                "summary": "Выход",
                "parameters": [{"in": "body", "name": "body", "schema": {"$ref": "#/definitions/RefreshRequest"}}],
                "responses": {"204": {"description": "No Content"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/users/me": {
            "get": {
                "security": [{"BearerAuth": []}],
                "tags": ["users"],
                "summary": "Профиль текущего пользователя",
                "responses": {"200": {"description": "OK"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/categories": {
            "get": {"tags": ["categories"], "summary": "Список категорий", "responses": {"200": {"description": "OK"}}},
            "post": {
                "security": [{"BearerAuth": []}],
                "tags": ["categories"],
                "summary": "Создать категорию",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/CreateCategoryRequest"}}],
                "responses": {"201": {"description": "Created"}, "400": {"description": "Bad Request"}}
            }
        },
        "/tags": {
            "get": {"tags": ["tags"], "summary": "Список тегов", "responses": {"200": {"description": "OK"}}},
            "post": {
                "security": [{"BearerAuth": []}],
                "tags": ["tags"],
                "summary": "Создать тег",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/CreateTagRequest"}}],
                "responses": {"201": {"description": "Created"}, "400": {"description": "Bad Request"}}
            }
        },
        "/decks": {
            "get": {"security": [{"BearerAuth": []}], "tags": ["decks"], "summary": "Мои наборы", "responses": {"200": {"description": "OK"}}},
            "post": {
                "security": [{"BearerAuth": []}],
                "tags": ["decks"],
                "summary": "Создать набор карточек",
                "parameters": [{"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/CreateDeckRequest"}}],
                "responses": {"201": {"description": "Created"}, "400": {"description": "Bad Request"}}
            }
        },
        "/decks/public": {
            "get": {"tags": ["decks"], "summary": "Публичные наборы", "parameters": [{"name": "limit", "in": "query", "type": "integer"}, {"name": "offset", "in": "query", "type": "integer"}], "responses": {"200": {"description": "OK"}}
            }
        },
        "/decks/{id}": {
            "get": {"security": [{"BearerAuth": []}], "tags": ["decks"], "summary": "Получить набор по ID", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}], "responses": {"200": {"description": "OK"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}},
            "put": {"security": [{"BearerAuth": []}], "tags": ["decks"], "summary": "Обновить набор", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}, {"in": "body", "name": "body", "schema": {"$ref": "#/definitions/UpdateDeckRequest"}}], "responses": {"200": {"description": "OK"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}},
            "delete": {"security": [{"BearerAuth": []}], "tags": ["decks"], "summary": "Удалить набор", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}], "responses": {"204": {"description": "No Content"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}}
        },
        "/decks/{deck_id}/cards": {
            "get": {"security": [{"BearerAuth": []}], "tags": ["cards"], "summary": "Карточки набора", "parameters": [{"name": "deck_id", "in": "path", "required": true, "type": "integer"}], "responses": {"200": {"description": "OK"}, "403": {"description": "Forbidden"}}},
            "post": {"security": [{"BearerAuth": []}], "tags": ["cards"], "summary": "Создать карточку", "parameters": [{"name": "deck_id", "in": "path", "required": true, "type": "integer"}, {"in": "body", "name": "body", "required": true, "schema": {"$ref": "#/definitions/CreateCardRequest"}}], "responses": {"201": {"description": "Created"}, "403": {"description": "Forbidden"}}}
        },
        "/cards/{id}": {
            "get": {"security": [{"BearerAuth": []}], "tags": ["cards"], "summary": "Получить карточку по ID", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}], "responses": {"200": {"description": "OK"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}},
            "put": {"security": [{"BearerAuth": []}], "tags": ["cards"], "summary": "Обновить карточку", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}, {"in": "body", "name": "body", "schema": {"$ref": "#/definitions/UpdateCardRequest"}}], "responses": {"200": {"description": "OK"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}},
            "delete": {"security": [{"BearerAuth": []}], "tags": ["cards"], "summary": "Удалить карточку", "parameters": [{"name": "id", "in": "path", "required": true, "type": "integer"}], "responses": {"204": {"description": "No Content"}, "403": {"description": "Forbidden"}, "404": {"description": "Not Found"}}}
        }
    },
    "securityDefinitions": {
        "BearerAuth": {"type": "apiKey", "name": "Authorization", "in": "header"}
    },
    "definitions": {
        "RegisterRequest": {"type": "object", "properties": {"email": {"type": "string"}, "password": {"type": "string"}, "username": {"type": "string"}}},
        "LoginRequest": {"type": "object", "properties": {"email": {"type": "string"}, "password": {"type": "string"}}},
        "RefreshRequest": {"type": "object", "properties": {"refresh_token": {"type": "string"}}},
        "TokenResponse": {"type": "object", "properties": {"access_token": {"type": "string"}, "refresh_token": {"type": "string"}, "expires_in": {"type": "integer"}, "token_type": {"type": "string"}}},
        "CreateCategoryRequest": {"type": "object", "properties": {"name": {"type": "string"}}},
        "CreateTagRequest": {"type": "object", "properties": {"name": {"type": "string"}}},
        "CreateDeckRequest": {"type": "object", "properties": {"title": {"type": "string"}, "description": {"type": "string"}, "category_id": {"type": "integer"}, "is_public": {"type": "boolean"}, "tag_ids": {"type": "array", "items": {"type": "integer"}}}},
        "UpdateDeckRequest": {"type": "object", "properties": {"title": {"type": "string"}, "description": {"type": "string"}, "category_id": {"type": "integer"}, "is_public": {"type": "boolean"}, "tag_ids": {"type": "array", "items": {"type": "integer"}}}},
        "CreateCardRequest": {"type": "object", "properties": {"question": {"type": "string"}, "answer": {"type": "string"}, "category_id": {"type": "integer"}, "tag_ids": {"type": "array", "items": {"type": "integer"}}}},
        "UpdateCardRequest": {"type": "object", "properties": {"question": {"type": "string"}, "answer": {"type": "string"}, "category_id": {"type": "integer"}, "tag_ids": {"type": "array", "items": {"type": "integer"}}}}
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api",
	Schemes:          []string{},
	Title:            "МозгоЁмка API",
	Description:      "REST API для приложения карточек МозгоЁмка",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}

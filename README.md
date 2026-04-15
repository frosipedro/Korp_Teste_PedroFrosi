# 🧾 Sistema de Notas Fiscais

Sistema completo de gerenciamento de notas fiscais com arquitetura de microsserviços, desenvolvido como projeto técnico fullstack.

---

## 🏗️ Arquitetura

```
┌────────────────────────────────────────────────────────────────────┐
│                         Docker Compose                             │
│                                                                    │
│  Browser                                                           │
│    │                                                               │
│    ▼                                                               │
│  ┌──────────────────────────┐   /api/inventory  ┌──────────────┐   │
│  │ Frontend + Nginx         │──────────────────▶│ Inventory   │   │
│  │ Angular :4200 / Nginx    │                   │ Service      │   │
│  └────────────┬─────────────┘   /api/billing    │ :8081        │   │
│               │────────────────────────────────▶└──────┬───────┘  │
│               │                                         │          │
│               │                                         │          │
│               │                              ┌──────────▼──────┐   │
│               │                              │ Billing Service  │  │
│               │                              │ :8082            │  │
│               │                              └──────────┬──────┘   │
│               │                                         │          │
│               │                                         ▼          │
│               └─────────────────────────────▶ PostgreSQL :5432    │
│                                            ▲                       │
│                                            └─ Inventory + Billing  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

**Stack:**

- **Frontend:** Angular 21.2.x + Angular Material + RxJS
- **Backend:** Go 1.26 com Gin Framework (2 microsserviços)
- **Banco de dados:** PostgreSQL 16
- **IA:** Groq API (análise manual de notas fiscais)
- **Infraestrutura:** Docker + Docker Compose + Nginx

---

## 📁 Estrutura do Projeto

```
Korp_Teste_PedroFrosi/
├── docker-compose.yml
├── .env                          ← você cria (não está no repo)
├── .gitignore
├── README.md
│
├── services/
│   ├── inventory/                # Serviço de Estoque (Go)
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   ├── main.go
│   │   ├── db/postgres.go
│   │   ├── handlers/product_handler.go
│   │   ├── models/model.go
│   │   └── migrations/001_create_products.sql
│   │
│   └── billing/                  # Serviço de Faturamento (Go)
│       ├── Dockerfile
│       ├── go.mod
│       ├── main.go
│       ├── db/postgres.go
│       ├── handlers/invoice_handler.go
│       ├── models/models.go
│       ├── services/ai_analysis.go
│       └── migrations/001_create_invoices.sql
│
└── frontend/                     # Angular
    ├── Dockerfile
    ├── nginx.conf
    ├── angular.json
    ├── package.json
    ├── tsconfig.json
    └── src/
        ├── main.ts
        ├── index.html
        ├── styles.scss
        └── app/
            ├── app.component.ts/html
            ├── app.routes.ts
            ├── core/
            │   ├── interceptors/error.interceptor.ts
            │   └── services/
            │       ├── product.service.ts
            │       └── invoice.service.ts
            ├── shared/models/
            │   ├── product.model.ts
            │   └── invoice.model.ts
            └── features/
                ├── products/
                └── invoices/
```

---

## ✅ Pré-requisitos

Instale antes de qualquer coisa:

| Ferramenta     | Versão mínima | Link                                           |
| -------------- | ------------- | ---------------------------------------------- |
| Docker Desktop | Mais recente  | https://www.docker.com/products/docker-desktop |
| Git            | Qualquer      | https://git-scm.com                            |

> Node.js e Go **não são necessários** para rodar via Docker. Só instale se quiser rodar os serviços localmente sem Docker.

---

## 🚀 Como Rodar (via Docker — recomendado)

### 1. Clone o repositório

```bash
git clone https://github.com/PedroFrosi/Korp_Teste_PedroFrosi.git
cd Korp_Teste_PedroFrosi
```

> Se você baixou como **ZIP** pelo GitHub, a pasta vai se chamar `Korp_Teste_PedroFrosi-main`. Renomeie para `Korp_Teste_PedroFrosi` antes de continuar.

### 2. Crie o arquivo `.env`

Na raiz do projeto (mesma pasta do `docker-compose.yml`), crie um arquivo chamado `.env`:

```bash
# Linux / macOS
touch .env

# Windows (PowerShell)
New-Item .env
```

Abra o arquivo e adicione:

```env
GROQ_API_KEY=gsk_xxxxxxxxxxxxxxxx
```

> Obtenha sua chave em https://console.groq.com/keys. Sem ela, a funcionalidade de análise por IA não funciona, mas o restante do sistema funciona normalmente.

### 3. Suba o projeto

```bash
docker-compose up --build
```

Na primeira execução o Docker vai:

- Baixar as imagens base (Go, Node, Postgres, Nginx)
- Instalar dependências do Go e do Angular
- Compilar os serviços
- Executar as migrations do banco automaticamente

Isso pode levar **3 a 10 minutos** dependendo da sua conexão.

### 4. Acesse o sistema

| Serviço                | URL                          |
| ---------------------- | ---------------------------- |
| **Frontend (Angular)** | http://localhost:4200        |
| **Inventory API**      | http://localhost:8081        |
| **Billing API**        | http://localhost:8082        |
| **Health Inventory**   | http://localhost:8081/health |
| **Health Billing**     | http://localhost:8082/health |

### 5. Parar o projeto

```bash
# Para os containers sem apagar dados
docker-compose down

# Para os containers E apaga o banco (reset total)
docker-compose down -v
```

---

## 💻 Como Rodar Localmente (sem Docker)

Necessário: **Go 1.22+**, **Node.js 20+**, **PostgreSQL 16**.

### 1. Suba só o Postgres via Docker

```bash
docker-compose up postgres
```

### 2. Rode o Serviço de Estoque

```bash
cd services/inventory
go mod tidy
DB_HOST=localhost DB_PORT=5432 DB_USER=invoice_user \
DB_PASSWORD=invoice_pass DB_NAME=invoice_db PORT=8081 \
go run main.go
```

### 3. Rode o Serviço de Faturamento

```bash
cd services/billing
go mod tidy
DB_HOST=localhost DB_PORT=5432 DB_USER=invoice_user \
DB_PASSWORD=invoice_pass DB_NAME=invoice_db PORT=8082 \
INVENTORY_URL=http://localhost:8081 \
GROQ_API_KEY=gsk_xxx \
go run main.go
```

### 4. Rode o Frontend

```bash
cd frontend
npm install
npm start
```

Acesse em http://localhost:4200.

> Rodando local, as URLs dos services Angular já apontam para `localhost:8081` e `localhost:8082` por padrão. Via Docker, o Nginx faz o proxy automaticamente.

---

## 🧪 Como Testar

### Pelo Navegador

Acesse http://localhost:4200 e use a interface para:

- Cadastrar, editar e excluir produtos
- Criar notas fiscais com múltiplos produtos
- Imprimir notas (muda status para Fechada e deduz estoque)
- Analisar a nota com IA sob demanda, sem alterar automaticamente produtos ou descrições

### Pelo Postman (ou qualquer cliente HTTP)

#### Produtos

```http
# Criar produto
POST http://localhost:8081/products
Content-Type: application/json

{
  "code": "PROD-001",
  "description": "Notebook Dell Inspiron",
  "balance": 10
}
```

```http
# Listar produtos
GET http://localhost:8081/products
```

```http
# Buscar por ID
GET http://localhost:8081/products/1
```

```http
# Atualizar produto
PUT http://localhost:8081/products/1
Content-Type: application/json

{
  "description": "Notebook Dell Inspiron 15",
  "balance": 20
}
```

```http
# Deletar produto
DELETE http://localhost:8081/products/1
```

#### Estoque

```http
# Deduzir estoque (com lock otimista)
POST http://localhost:8081/stock/deduct
Content-Type: application/json

{
  "product_id": 1,
  "quantity": 3
}
```

#### Notas Fiscais

```http
# Criar nota fiscal
POST http://localhost:8082/invoices
Content-Type: application/json

{
  "items": [
    {
      "product_id": 1,
      "product_code": "PROD-001",
      "description": "Notebook Dell Inspiron",
      "quantity": 2
    }
  ]
}
```

```http
# Listar notas
GET http://localhost:8082/invoices
```

```http
# Buscar nota por ID (com itens)
GET http://localhost:8082/invoices/1
```

```http
# Imprimir nota (fecha e deduz estoque)
POST http://localhost:8082/invoices/1/print
Content-Type: application/json

{
  "idempotency_key": "550e8400-e29b-41d4-a716-446655440000"
}
```

> O `idempotency_key` deve ser um UUID único por impressão. Gere um em https://www.uuidgenerator.net. Reenviar a mesma chave retorna sucesso sem reprocessar.

#### IA — Análise de Nota Fiscal

```http
POST http://localhost:8082/ai/analyze
Content-Type: application/json

{
  "context": "materiais de escritório para a matriz",
  "items": [
    {
      "product_id": 1,
      "product_code": "PROD-001",
      "description": "Notebook Dell Inspiron",
      "quantity": 2
    }
  ]
}
```

---

## 🔍 Funcionalidades Técnicas

### Lock Otimista (Estoque)

O endpoint `POST /stock/deduct` usa a coluna `version` da tabela `products`. Se dois processos tentarem deduzir ao mesmo tempo, apenas um vence — o outro recebe `409 Conflict` com mensagem para retry.

### Idempotência (Impressão)

O campo `idempotency_key` na tabela `invoices` tem constraint `UNIQUE`. Reenviar a mesma chave para `POST /invoices/:id/print` retorna o resultado original sem reprocessar, protegendo contra duplo clique ou retry de rede.

### Retry com Backoff (Impressão)

O serviço de billing tenta deduzir o estoque até **3 vezes** com intervalo crescente (200ms, 400ms, 600ms) antes de retornar erro ao frontend.

### Interceptor de Erros (Frontend)

Todas as chamadas HTTP passam pelo `ErrorInterceptor`, que captura erros e exibe um `MatSnackBar` com a mensagem vinda do backend (campo `error`) ou uma mensagem padrão por status HTTP.

### Análise por IA

Na criação de uma nova nota fiscal, o usuário pode clicar em **Analisar com IA** para enviar o rascunho ao endpoint `/ai/analyze`. A IA retorna um resumo em pt-BR, categoria sugerida, nível de risco, alertas e recomendações. A análise é manual e não altera produtos, quantidades ou descrições.

---

## 🗄️ Banco de Dados

As migrations rodam automaticamente ao subir o Docker pela primeira vez. As tabelas criadas são:

**`products`**
| Coluna | Tipo | Descrição |
|---|---|---|
| id | SERIAL | Chave primária |
| code | VARCHAR(50) | Código único do produto |
| description | VARCHAR(255) | Descrição |
| balance | INTEGER | Saldo em estoque |
| version | INTEGER | Controle de lock otimista |
| created_at | TIMESTAMPTZ | Data de criação |
| updated_at | TIMESTAMPTZ | Data de atualização |

**`invoices`**
| Coluna | Tipo | Descrição |
|---|---|---|
| id | SERIAL | Chave primária |
| number | INTEGER | Número sequencial (inicia em 1000) |
| status | ENUM | `open` ou `closed` |
| idempotency_key | VARCHAR(100) | Chave única de impressão |
| closed_at | TIMESTAMPTZ | Data e hora em que a nota foi fechada |
| created_at | TIMESTAMPTZ | Data de criação |
| updated_at | TIMESTAMPTZ | Data de atualização |

**`invoice_items`**
| Coluna | Tipo | Descrição |
|---|---|---|
| id | SERIAL | Chave primária |
| invoice_id | INTEGER | FK para invoices |
| product_id | INTEGER | ID do produto |
| product_code | VARCHAR(50) | Código do produto |
| description | VARCHAR(255) | Descrição |
| quantity | INTEGER | Quantidade |

---

## ⚠️ Problemas Comuns

| Erro                            | Causa                              | Solução                                       |
| ------------------------------- | ---------------------------------- | --------------------------------------------- |
| `cannot find module` (Go)       | `go.sum` ausente                   | `cd services/inventory && go mod tidy`        |
| Angular não compila             | `npm install` não rodou            | `cd frontend && npm install`                  |
| Porta 5432 ocupada              | Postgres local rodando             | `sudo lsof -i :5432` e pare o processo        |
| Porta 4200/8081/8082 ocupada    | Outro processo usando              | Troque a porta no `docker-compose.yml`        |
| IA retorna erro 500             | `GROQ_API_KEY` inválida ou ausente | Verifique o `.env` na raiz                    |
| Frontend não acessa o backend   | URL errada nos services            | Verifique se está rodando via Docker ou local |
| Banco vazio após reiniciar      | Volume foi removido com `-v`       | Normal — migrations recriam as tabelas        |
| `connection refused` no billing | Inventory ainda não subiu          | Aguarde, o `depends_on` garante a ordem       |

---

## 📄 Licença

Projeto desenvolvido para fins de avaliação técnica.

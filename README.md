# Backend Go

IA parcialmente integrada seria a OpenIa só para fazer demonstração sem framework de suporte clean

" para fazer a integração com gemini ou a openia coloque uma chave de Api externa e aplique no powershell  

ainda prototipon de intregar com as APIS de IA realmente para funcionar teria que ter uma disponivel a uso todos que forem executar ou testar tente pegar a do Google mais facil de achar a APi para teste ou uso sla o nome do studio e (Google AI Studio)

Exemplo pequeno de API em Go usando `net/http`, sem framework ou dependências externas.

## Requisitos

- Go 1.22 ou superior
- Uma chave de API da OpenAI

## Executar no PowerShell

```powershell
$env:OPENAI_API_KEY = "sua-chave"
go run .
```

Opcionalmente, configure `OPENAI_MODEL` (padrão: `gpt-4.1-mini`) e `PORT` (padrão: `8080`).

## Endpoints

- `GET /health` — verifica se o servidor está ativo.
- `POST /chat` — envia uma mensagem para a IA.

Exemplo:

```powershell
Invoke-RestMethod -Uri http://localhost:8080/chat `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"message":"Explique o que é Go em uma frase."}'
```

A resposta tem o formato `{"response":"..."}`. A chave é lida da variável de ambiente e não deve ser enviada pelo cliente nem incluída no código.

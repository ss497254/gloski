import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Snippet {
  id: string
  title: string
  code: string
  language: string
  description?: string
  tags: string[]
  favorite: boolean
  createdAt: string
}

interface SnippetsState {
  snippets: Snippet[]
  addSnippet: (snippet: Omit<Snippet, 'id' | 'createdAt' | 'favorite'>) => void
  updateSnippet: (id: string, data: Partial<Snippet>) => void
  deleteSnippet: (id: string) => void
  toggleFavorite: (id: string) => void
}

// Mock data with various languages
const initialSnippets: Snippet[] = [
  {
    id: '1',
    title: 'React useDebounce Hook',
    code: `import { useState, useEffect } from 'react'

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => clearTimeout(timer)
  }, [value, delay])

  return debouncedValue
}`,
    language: 'typescript',
    description: 'A custom React hook for debouncing values',
    tags: ['react', 'hooks', 'debounce'],
    favorite: true,
    createdAt: '2024-01-15T10:00:00Z',
  },
  {
    id: '2',
    title: 'Python FastAPI Endpoint',
    code: `from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI()

class Item(BaseModel):
    name: str
    price: float
    is_available: bool = True

@app.post("/items/")
async def create_item(item: Item):
    if item.price < 0:
        raise HTTPException(status_code=400, detail="Price must be positive")
    return {"item": item, "message": "Item created successfully"}`,
    language: 'python',
    description: 'Basic FastAPI endpoint with Pydantic validation',
    tags: ['python', 'fastapi', 'api'],
    favorite: true,
    createdAt: '2024-01-14T09:00:00Z',
  },
  {
    id: '3',
    title: 'Go HTTP Server',
    code: `package main

import (
    "encoding/json"
    "log"
    "net/http"
)

type Response struct {
    Message string \`json:"message"\`
    Status  int    \`json:"status"\`
}

func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    response := Response{Message: "Hello, World!", Status: 200}
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/", handler)
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}`,
    language: 'go',
    description: 'Simple Go HTTP server with JSON response',
    tags: ['go', 'http', 'server'],
    favorite: false,
    createdAt: '2024-01-13T14:00:00Z',
  },
  {
    id: '4',
    title: 'SQL User Query',
    code: `-- Get users with their order counts
SELECT 
    u.id,
    u.name,
    u.email,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total), 0) as total_spent
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
GROUP BY u.id, u.name, u.email
HAVING order_count > 0
ORDER BY total_spent DESC
LIMIT 10;`,
    language: 'sql',
    description: 'Query to get top users by spending in last 30 days',
    tags: ['sql', 'query', 'analytics'],
    favorite: false,
    createdAt: '2024-01-12T11:00:00Z',
  },
  {
    id: '5',
    title: 'Bash Deploy Script',
    code: `#!/bin/bash
set -e

# Configuration
APP_NAME="myapp"
DEPLOY_DIR="/var/www/$APP_NAME"
BACKUP_DIR="/var/backups/$APP_NAME"

echo "Starting deployment..."

# Create backup
timestamp=$(date +%Y%m%d_%H%M%S)
echo "Creating backup..."
tar -czf "$BACKUP_DIR/backup_$timestamp.tar.gz" "$DEPLOY_DIR"

# Pull latest changes
cd "$DEPLOY_DIR"
git pull origin main

# Install dependencies
npm ci --production

# Restart service
sudo systemctl restart "$APP_NAME"

echo "Deployment completed successfully!"`,
    language: 'bash',
    description: 'Simple deployment script with backup',
    tags: ['bash', 'deploy', 'devops'],
    favorite: true,
    createdAt: '2024-01-11T16:00:00Z',
  },
  {
    id: '6',
    title: 'CSS Flexbox Center',
    code: `.center-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}

/* Alternative with grid */
.center-grid {
  display: grid;
  place-items: center;
  min-height: 100vh;
}`,
    language: 'css',
    description: 'Different ways to center content',
    tags: ['css', 'flexbox', 'grid'],
    favorite: false,
    createdAt: '2024-01-10T08:00:00Z',
  },
  {
    id: '7',
    title: 'JavaScript Array Utils',
    code: `// Remove duplicates
const unique = (arr) => [...new Set(arr)]

// Group by key
const groupBy = (arr, key) => 
  arr.reduce((acc, item) => {
    const group = item[key]
    acc[group] = acc[group] || []
    acc[group].push(item)
    return acc
  }, {})

// Chunk array
const chunk = (arr, size) => 
  Array.from({ length: Math.ceil(arr.length / size) }, (_, i) =>
    arr.slice(i * size, i * size + size)
  )

// Shuffle array
const shuffle = (arr) => 
  [...arr].sort(() => Math.random() - 0.5)`,
    language: 'javascript',
    description: 'Common array utility functions',
    tags: ['javascript', 'array', 'utils'],
    favorite: true,
    createdAt: '2024-01-09T13:00:00Z',
  },
  {
    id: '8',
    title: 'Docker Compose Stack',
    code: `version: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - DATABASE_URL=postgres://user:pass@db:5432/mydb
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=mydb

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:`,
    language: 'yaml',
    description: 'Docker Compose setup with app, database, and cache',
    tags: ['docker', 'compose', 'devops'],
    favorite: false,
    createdAt: '2024-01-08T10:00:00Z',
  },
  {
    id: '9',
    title: 'Rust Error Handling',
    code: `use std::fs::File;
use std::io::{self, Read};

fn read_file_contents(path: &str) -> Result<String, io::Error> {
    let mut file = File::open(path)?;
    let mut contents = String::new();
    file.read_to_string(&mut contents)?;
    Ok(contents)
}

fn main() {
    match read_file_contents("config.txt") {
        Ok(contents) => println!("File contents: {}", contents),
        Err(e) => eprintln!("Error reading file: {}", e),
    }
}`,
    language: 'rust',
    description: 'Idiomatic Rust error handling with Result',
    tags: ['rust', 'error-handling'],
    favorite: false,
    createdAt: '2024-01-07T15:00:00Z',
  },
  {
    id: '10',
    title: 'TypeScript Zod Schema',
    code: `import { z } from 'zod'

const UserSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  name: z.string().min(2).max(50),
  age: z.number().int().positive().optional(),
  role: z.enum(['admin', 'user', 'guest']),
  metadata: z.record(z.string()).optional(),
  createdAt: z.coerce.date(),
})

type User = z.infer<typeof UserSchema>

// Usage
const result = UserSchema.safeParse(data)
if (result.success) {
  const user: User = result.data
} else {
  console.error(result.error.issues)
}`,
    language: 'typescript',
    description: 'Zod schema for runtime type validation',
    tags: ['typescript', 'zod', 'validation'],
    favorite: true,
    createdAt: '2024-01-06T09:00:00Z',
  },
]

function generateId(): string {
  return Math.random().toString(36).substring(2, 10)
}

export const useSnippetsStore = create<SnippetsState>()(
  persist(
    (set) => ({
      snippets: initialSnippets,

      addSnippet: (snippet) =>
        set((state) => ({
          snippets: [
            {
              ...snippet,
              id: generateId(),
              favorite: false,
              createdAt: new Date().toISOString(),
            },
            ...state.snippets,
          ],
        })),

      updateSnippet: (id, data) =>
        set((state) => ({
          snippets: state.snippets.map((s) =>
            s.id === id ? { ...s, ...data } : s
          ),
        })),

      deleteSnippet: (id) =>
        set((state) => ({
          snippets: state.snippets.filter((s) => s.id !== id),
        })),

      toggleFavorite: (id) =>
        set((state) => ({
          snippets: state.snippets.map((s) =>
            s.id === id ? { ...s, favorite: !s.favorite } : s
          ),
        })),
    }),
    {
      name: 'gloski-snippets',
    }
  )
)

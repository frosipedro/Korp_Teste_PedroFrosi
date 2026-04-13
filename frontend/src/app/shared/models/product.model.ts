export interface Product {
  id: number
  code: string
  description: string
  balance: number
  version: number
  created_at: string
  updated_at: string
}

export interface CreateProductRequest {
  code: string
  description: string
  balance: number
}

export interface UpdateProductRequest {
  description?: string
  balance?: number
}

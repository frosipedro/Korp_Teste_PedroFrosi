export type InvoiceStatus = 'open' | 'closed'

export interface InvoiceItem {
  id: number
  invoice_id: number
  product_id: number
  product_code: string
  description: string
  quantity: number
  created_at: string
}

export interface Invoice {
  id: number
  number: number
  status: InvoiceStatus
  idempotency_key?: string
  items?: InvoiceItem[]
  created_at: string
  updated_at: string
}

export interface CreateInvoiceItemRequest {
  product_id: number
  product_code: string
  description: string
  quantity: number
}

export interface CreateInvoiceRequest {
  items: CreateInvoiceItemRequest[]
}

export interface PrintInvoiceRequest {
  idempotency_key: string
}

export interface PrintInvoiceResponse {
  invoice_id: number
  invoice_number: number
  status: string
  message: string
}

export interface AISuggestionResponse {
  suggestions: string[]
}

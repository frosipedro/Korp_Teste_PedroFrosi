import { Injectable } from '@angular/core'
import { HttpClient } from '@angular/common/http'
import { Observable } from 'rxjs'
import {
  Invoice,
  CreateInvoiceRequest,
  PrintInvoiceRequest,
  PrintInvoiceResponse,
  AISuggestionResponse,
} from '../../shared/models/invoice.model'

@Injectable({ providedIn: 'root' })
export class InvoiceService {
  private readonly baseUrl = '/api/billing/invoices'
  private readonly aiUrl = '/api/billing/ai/suggest'

  constructor(private http: HttpClient) {}

  list(): Observable<Invoice[]> {
    return this.http.get<Invoice[]>(this.baseUrl)
  }

  getById(id: number): Observable<Invoice> {
    return this.http.get<Invoice>(`${this.baseUrl}/${id}`)
  }

  create(payload: CreateInvoiceRequest): Observable<Invoice> {
    return this.http.post<Invoice>(this.baseUrl, payload)
  }

  print(
    id: number,
    payload: PrintInvoiceRequest,
  ): Observable<PrintInvoiceResponse> {
    return this.http.post<PrintInvoiceResponse>(
      `${this.baseUrl}/${id}/print`,
      payload,
    )
  }

  suggest(description: string): Observable<AISuggestionResponse> {
    return this.http.post<AISuggestionResponse>(this.aiUrl, { description })
  }
}

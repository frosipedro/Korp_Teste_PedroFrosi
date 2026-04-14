import { Injectable } from '@angular/core'
import { HttpClient, HttpContext } from '@angular/common/http'
import { Observable } from 'rxjs'
import {
  Invoice,
  CreateInvoiceRequest,
  PrintInvoiceRequest,
  PrintInvoiceResponse,
  AISuggestionResponse,
} from '../../shared/models/invoice.model'
import { SUPPRESS_GLOBAL_ERROR_SNACKBAR } from '../http/request-flags'

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
    return this.http.post<Invoice>(this.baseUrl, payload, {
      context: new HttpContext().set(SUPPRESS_GLOBAL_ERROR_SNACKBAR, true),
    })
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
    return this.http.post<AISuggestionResponse>(
      this.aiUrl,
      { description },
      {
        context: new HttpContext().set(SUPPRESS_GLOBAL_ERROR_SNACKBAR, true),
      },
    )
  }
}

import { Injectable } from '@angular/core'
import { HttpClient, HttpContext } from '@angular/common/http'
import { Observable } from 'rxjs'
import {
  Product,
  CreateProductRequest,
  UpdateProductRequest,
} from '../../shared/models/product.model'
import { SUPPRESS_GLOBAL_ERROR_SNACKBAR } from '../http/request-flags'

@Injectable({ providedIn: 'root' })
export class ProductService {
  private readonly baseUrl = '/api/inventory/products'

  constructor(private http: HttpClient) {}

  list(suppressGlobalErrorSnackbar = false): Observable<Product[]> {
    if (suppressGlobalErrorSnackbar) {
      return this.http.get<Product[]>(this.baseUrl, {
        context: new HttpContext().set(SUPPRESS_GLOBAL_ERROR_SNACKBAR, true),
      })
    }

    return this.http.get<Product[]>(this.baseUrl)
  }

  getById(id: number): Observable<Product> {
    return this.http.get<Product>(`${this.baseUrl}/${id}`)
  }

  create(payload: CreateProductRequest): Observable<Product> {
    return this.http.post<Product>(this.baseUrl, payload, {
      context: new HttpContext().set(SUPPRESS_GLOBAL_ERROR_SNACKBAR, true),
    })
  }

  update(id: number, payload: UpdateProductRequest): Observable<Product> {
    return this.http.put<Product>(`${this.baseUrl}/${id}`, payload, {
      context: new HttpContext().set(SUPPRESS_GLOBAL_ERROR_SNACKBAR, true),
    })
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`)
  }
}

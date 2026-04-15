import { Component, OnDestroy, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import { RouterLink } from '@angular/router'
import { HttpErrorResponse } from '@angular/common/http'
import { MatTableModule } from '@angular/material/table'
import { MatButtonModule } from '@angular/material/button'
import { MatButtonToggleModule } from '@angular/material/button-toggle'
import { MatIconModule } from '@angular/material/icon'
import { MatFormFieldModule } from '@angular/material/form-field'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { MatSelectModule } from '@angular/material/select'
import { MatChipsModule } from '@angular/material/chips'
import { BehaviorSubject, finalize } from 'rxjs'
import { ProductService } from '../../core/services/product.service'
import { HttpErrorService } from '../../core/services/http-error.service'
import { Product } from '../../shared/models/product.model'

type SortDirection = 'asc' | 'desc'
type ProductSortField =
  | 'code'
  | 'description'
  | 'balance'
  | 'created_at'
  | 'updated_at'

@Component({
  selector: 'app-product-list',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MatTableModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatIconModule,
    MatFormFieldModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    MatSelectModule,
    MatChipsModule,
  ],
  templateUrl: './product-list.component.html',
})
export class ProductListComponent implements OnInit, OnDestroy {
  products: Product[] = []
  loading$ = new BehaviorSubject<boolean>(false)
  loadError: string | null = null
  retryInSeconds: number | null = null
  displayedColumns = ['code', 'description', 'balance', 'actions']
  sortField: ProductSortField = 'code'
  sortDirection: SortDirection = 'asc'
  private retryIntervalId: ReturnType<typeof setInterval> | null = null
  private readonly autoRetryDelaySeconds = 8

  readonly sortOptions = [
    { value: 'code', label: 'Código' },
    { value: 'description', label: 'Descrição' },
    { value: 'balance', label: 'Saldo' },
    { value: 'created_at', label: 'Criada em' },
    { value: 'updated_at', label: 'Atualizada em' },
  ] as const

  constructor(
    private productService: ProductService,
    private snackBar: MatSnackBar,
    private httpErrorService: HttpErrorService,
  ) {}

  ngOnInit(): void {
    this.load()
  }

  ngOnDestroy(): void {
    this.clearScheduledRetry()
  }

  load(): void {
    this.clearScheduledRetry()
    this.loadError = null

    this.loading$.next(true)
    this.productService
      .list(true)
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe({
        next: (data) => {
          this.products = data
          this.loadError = null
        },
        error: (error: unknown) => {
          this.products = []
          this.loadError = this.httpErrorService.getMessage(error)

          if (this.shouldScheduleRetry(error)) {
            this.scheduleRetry()
          }
        },
      })
  }

  retryNow(): void {
    this.load()
  }

  get sortedProducts(): Product[] {
    const directionFactor = this.sortDirection === 'asc' ? 1 : -1

    return [...this.products].sort((left, right) => {
      let comparison = 0

      switch (this.sortField) {
        case 'code':
          comparison = this.compareText(left.code, right.code)
          break
        case 'description':
          comparison = this.compareText(left.description, right.description)
          break
        case 'balance':
          comparison = left.balance - right.balance
          break
        case 'created_at':
          comparison = this.compareDate(left.created_at, right.created_at)
          break
        case 'updated_at':
          comparison = this.compareDate(left.updated_at, right.updated_at)
          break
      }

      return comparison * directionFactor
    })
  }

  setSortField(field: ProductSortField): void {
    this.sortField = field
  }

  setSortDirection(direction: SortDirection): void {
    this.sortDirection = direction
  }

  delete(id: number): void {
    if (!confirm('Confirmar exclusão do produto?')) return
    this.productService.delete(id).subscribe({
      next: () => {
        this.snackBar.open('Produto excluído.', 'Fechar', { duration: 3000 })
        this.load()
      },
    })
  }

  private compareText(left: string, right: string): number {
    return left.localeCompare(right, 'pt-BR', {
      numeric: true,
      sensitivity: 'base',
    })
  }

  private compareDate(left: string, right: string): number {
    return this.compareNumber(
      this.parseTimestamp(left),
      this.parseTimestamp(right),
    )
  }

  private compareNumber(left: number, right: number): number {
    return left - right
  }

  private parseTimestamp(value: string): number {
    const timestamp = Date.parse(value)
    return Number.isNaN(timestamp) ? 0 : timestamp
  }

  private shouldScheduleRetry(error: unknown): boolean {
    return (
      error instanceof HttpErrorResponse &&
      [0, 502, 503, 504].includes(error.status)
    )
  }

  private scheduleRetry(): void {
    this.clearScheduledRetry()

    let remaining = this.autoRetryDelaySeconds
    this.retryInSeconds = remaining

    this.retryIntervalId = setInterval(() => {
      remaining -= 1
      this.retryInSeconds = remaining

      if (remaining <= 0) {
        this.clearScheduledRetry()
        this.load()
      }
    }, 1000)
  }

  private clearScheduledRetry(): void {
    if (this.retryIntervalId !== null) {
      clearInterval(this.retryIntervalId)
      this.retryIntervalId = null
    }

    this.retryInSeconds = null
  }
}

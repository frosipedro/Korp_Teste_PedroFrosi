import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import { RouterLink } from '@angular/router'
import { MatTableModule } from '@angular/material/table'
import { MatButtonModule } from '@angular/material/button'
import { MatButtonToggleModule } from '@angular/material/button-toggle'
import { MatIconModule } from '@angular/material/icon'
import { MatFormFieldModule } from '@angular/material/form-field'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { MatChipsModule } from '@angular/material/chips'
import { MatTooltipModule } from '@angular/material/tooltip'
import { MatSelectModule } from '@angular/material/select'
import { BehaviorSubject, finalize } from 'rxjs'
import { InvoiceService } from '../../core/services/invoice.service'
import { Invoice, InvoiceStatus } from '../../shared/models/invoice.model'
import { v4 as uuidv4 } from 'uuid'

type SortDirection = 'asc' | 'desc'
type InvoiceSortField =
  | 'number'
  | 'status'
  | 'created_at'
  | 'closed_at'
  | 'updated_at'

@Component({
  selector: 'app-invoice-list',
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
    MatChipsModule,
    MatTooltipModule,
    MatSelectModule,
  ],
  templateUrl: './invoice-list.component.html',
})
export class InvoiceListComponent implements OnInit {
  invoices: Invoice[] = []
  loading$ = new BehaviorSubject<boolean>(false)
  printingId$ = new BehaviorSubject<number | null>(null)
  displayedColumns = ['number', 'status', 'created_at', 'closed_at', 'actions']
  sortField: InvoiceSortField = 'number'
  sortDirection: SortDirection = 'desc'

  readonly sortOptions = [
    { value: 'number', label: 'Número' },
    { value: 'status', label: 'Status' },
    { value: 'created_at', label: 'Criada em' },
    { value: 'closed_at', label: 'Fechada em' },
    { value: 'updated_at', label: 'Atualizada em' },
  ] as const

  constructor(
    private invoiceService: InvoiceService,
    private snackBar: MatSnackBar,
  ) {}

  ngOnInit(): void {
    this.load()
  }

  load(): void {
    this.loading$.next(true)
    this.invoiceService
      .list()
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe((data) => (this.invoices = data))
  }

  get sortedInvoices(): Invoice[] {
    const directionFactor = this.sortDirection === 'asc' ? 1 : -1

    return [...this.invoices].sort((left, right) => {
      let comparison = 0

      switch (this.sortField) {
        case 'number':
          comparison = left.number - right.number
          break
        case 'status':
          comparison = this.compareNumber(
            this.statusRank(left.status),
            this.statusRank(right.status),
          )
          break
        case 'created_at':
          comparison = this.compareDate(left.created_at, right.created_at)
          break
        case 'closed_at':
          comparison = this.compareDate(
            left.closed_at ?? '',
            right.closed_at ?? '',
          )
          break
        case 'updated_at':
          comparison = this.compareDate(left.updated_at, right.updated_at)
          break
      }

      return comparison * directionFactor
    })
  }

  setSortField(field: InvoiceSortField): void {
    this.sortField = field
  }

  setSortDirection(direction: SortDirection): void {
    this.sortDirection = direction
  }

  print(invoice: Invoice): void {
    if (invoice.status === 'closed') return

    this.printingId$.next(invoice.id)
    const idempotencyKey = uuidv4()

    this.invoiceService
      .print(invoice.id, { idempotency_key: idempotencyKey })
      .pipe(finalize(() => this.printingId$.next(null)))
      .subscribe({
        next: (res) => {
          this.snackBar.open(res.message, 'Fechar', { duration: 4000 })
          this.load()
        },
        error: () => {
          this.printingId$.next(null)
        },
      })
  }

  isPrinting(id: number): boolean {
    return this.printingId$.value === id
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
    if (!value) {
      return 0
    }

    const timestamp = Date.parse(value)
    return Number.isNaN(timestamp) ? 0 : timestamp
  }

  private statusRank(status: InvoiceStatus): number {
    switch (status) {
      case 'open':
        return 0
      case 'closed':
        return 1
      default:
        return 2
    }
  }
}

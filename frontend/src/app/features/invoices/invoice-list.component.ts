import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import { RouterLink } from '@angular/router'
import { MatTableModule } from '@angular/material/table'
import { MatButtonModule } from '@angular/material/button'
import { MatIconModule } from '@angular/material/icon'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { MatChipsModule } from '@angular/material/chips'
import { MatTooltipModule } from '@angular/material/tooltip'
import { BehaviorSubject, finalize } from 'rxjs'
import { InvoiceService } from '../../core/services/invoice.service'
import { Invoice } from '../../shared/models/invoice.model'
import { v4 as uuidv4 } from 'uuid'

@Component({
  selector: 'app-invoice-list',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    MatChipsModule,
    MatTooltipModule,
  ],
  templateUrl: './invoice-list.component.html',
})
export class InvoiceListComponent implements OnInit {
  invoices: Invoice[] = []
  loading$ = new BehaviorSubject<boolean>(false)
  printingId$ = new BehaviorSubject<number | null>(null)
  displayedColumns = ['number', 'status', 'created_at', 'actions']

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
      })
  }

  isPrinting(id: number): boolean {
    return this.printingId$.value === id
  }
}

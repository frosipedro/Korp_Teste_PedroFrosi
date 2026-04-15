import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import {
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  FormArray,
  AbstractControl,
  Validators,
} from '@angular/forms'
import { Router, RouterLink } from '@angular/router'
import { MatFormFieldModule } from '@angular/material/form-field'
import { MatInputModule } from '@angular/material/input'
import { MatButtonModule } from '@angular/material/button'
import { MatIconModule } from '@angular/material/icon'
import { MatTooltipModule } from '@angular/material/tooltip'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar } from '@angular/material/snack-bar'
import { MatDividerModule } from '@angular/material/divider'
import { MatChipsModule } from '@angular/material/chips'
import { MatCardModule } from '@angular/material/card'
import { BehaviorSubject, finalize } from 'rxjs'
import { InvoiceService } from '../../core/services/invoice.service'
import { ProductService } from '../../core/services/product.service'
import { Product } from '../../shared/models/product.model'
import {
  HttpErrorService,
  StockConflictDetails,
} from '../../core/services/http-error.service'
import { AIAnalysisResponse } from '../../shared/models/invoice.model'

@Component({
  selector: 'app-invoice-form',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    MatDividerModule,
    MatChipsModule,
    MatCardModule,
  ],
  templateUrl: './invoice-form.component.html',
})
export class InvoiceFormComponent implements OnInit {
  form!: FormGroup
  loading$ = new BehaviorSubject<boolean>(false)
  analyzing$ = new BehaviorSubject<boolean>(false)
  analysisResult: AIAnalysisResponse | null = null
  analysisMessage: string | null = null
  submissionError: string | null = null
  stockConflict: StockConflictDetails | null = null
  products: Product[] = []

  constructor(
    private fb: FormBuilder,
    private invoiceService: InvoiceService,
    private productService: ProductService,
    private router: Router,
    private snackBar: MatSnackBar,
    private httpErrorService: HttpErrorService,
  ) {}

  ngOnInit(): void {
    this.form = this.fb.group({
      analysisContext: [''],
      items: this.fb.array([]),
    })

    this.form.valueChanges.subscribe(() => {
      if (this.submissionError || this.stockConflict) {
        this.clearSubmissionError()
      }

      if (this.analysisResult || this.analysisMessage) {
        this.clearAnalysisState()
      }
    })

    this.productService.list().subscribe((p) => (this.products = p))
  }

  get items(): FormArray {
    return this.form.get('items') as FormArray
  }

  addItem(): void {
    this.items.push(
      this.fb.group({
        product_id: [null, Validators.required],
        product_code: [''],
        description: ['', [Validators.required, Validators.maxLength(255)]],
        quantity: [1, [Validators.required, Validators.min(1)]],
      }),
    )
  }

  removeItem(index: number): void {
    this.items.removeAt(index)
  }

  selectProduct(index: number, productId: number): void {
    const product = this.products.find((p) => p.id === productId)
    if (!product) return
    this.items.at(index).patchValue({
      product_id: product.id,
      product_code: product.code,
      description: product.description,
    })
  }

  analyze(): void {
    this.clearAnalysisState()

    if (this.items.length === 0) {
      this.analysisMessage = 'Adicione ao menos um produto antes de analisar.'
      return
    }

    if (this.form.invalid) {
      this.form.markAllAsTouched()
      this.analysisMessage = 'Corrija os campos destacados antes de analisar.'
      return
    }

    this.analyzing$.next(true)
    this.invoiceService
      .analyze({
        context: (this.form.get('analysisContext')?.value ?? '').trim(),
        items: this.items.getRawValue(),
      })
      .pipe(finalize(() => this.analyzing$.next(false)))
      .subscribe({
        next: (res) => {
          this.analysisResult = res
        },
        error: (error) => {
          this.analysisMessage = this.httpErrorService.getMessage(error)
        },
      })
  }

  submit(): void {
    this.clearSubmissionError()

    if (this.items.length === 0) {
      this.submissionError = 'Adicione ao menos um produto antes de salvar.'
      return
    }

    if (this.form.invalid) {
      this.form.markAllAsTouched()
      this.submissionError = 'Corrija os campos destacados antes de salvar.'
      return
    }

    this.loading$.next(true)
    this.invoiceService
      .create({ items: this.items.getRawValue() })
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe({
        next: () => {
          this.snackBar.open('Nota fiscal criada.', 'Fechar', {
            duration: 3000,
          })
          this.router.navigate(['/invoices'])
        },
        error: (error) => {
          this.loading$.next(false)
          const stockConflict =
            this.httpErrorService.extractStockConflictDetails(error)

          if (stockConflict) {
            this.stockConflict = stockConflict
            this.submissionError = this.buildStockConflictMessage(stockConflict)
            return
          }

          this.submissionError = this.httpErrorService.getMessage(error)
        },
      })
  }

  get totalQuantity(): number {
    return this.items.controls.reduce((total, item) => {
      return total + Number(item.get('quantity')?.value || 0)
    }, 0)
  }

  private clearSubmissionError(): void {
    this.submissionError = null
    this.stockConflict = null
  }

  private clearAnalysisState(): void {
    this.analysisResult = null
    this.analysisMessage = null
  }

  isStockConflictRow(item: AbstractControl): boolean {
    return (
      this.stockConflict !== null &&
      item.get('product_id')?.value === this.stockConflict.productId
    )
  }

  riskLabel(level: string | undefined): string {
    switch ((level ?? '').toLowerCase()) {
      case 'baixo':
        return 'Baixo'
      case 'medio':
        return 'Médio'
      case 'alto':
        return 'Alto'
      default:
        return level ? level : 'Indefinido'
    }
  }

  riskClass(level: string | undefined): string {
    switch ((level ?? '').toLowerCase()) {
      case 'baixo':
        return 'risk-chip--low'
      case 'medio':
        return 'risk-chip--medium'
      case 'alto':
        return 'risk-chip--high'
      default:
        return 'risk-chip--neutral'
    }
  }

  private buildStockConflictMessage(details: StockConflictDetails): string {
    const product = this.products.find((p) => p.id === details.productId)
    const label = product
      ? `${product.code} — ${product.description}`
      : `produto ID ${details.productId}`

    return `O ${label} não possui saldo suficiente. Solicitado ${details.requested}, disponível ${details.available}.`
  }
}

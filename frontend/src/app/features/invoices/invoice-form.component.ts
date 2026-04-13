import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import {
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  FormArray,
  Validators,
} from '@angular/forms'
import { Router, RouterLink } from '@angular/router'
import { MatFormFieldModule } from '@angular/material/form-field'
import { MatInputModule } from '@angular/material/input'
import { MatButtonModule } from '@angular/material/button'
import { MatIconModule } from '@angular/material/icon'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { MatDividerModule } from '@angular/material/divider'
import { MatListModule } from '@angular/material/list'
import { MatChipsModule } from '@angular/material/chips'
import {
  BehaviorSubject,
  finalize,
  debounceTime,
  distinctUntilChanged,
} from 'rxjs'
import { InvoiceService } from '../../core/services/invoice.service'
import { ProductService } from '../../core/services/product.service'
import { Product } from '../../shared/models/product.model'

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
    MatProgressSpinnerModule,
    MatSnackBarModule,
    MatDividerModule,
    MatListModule,
    MatChipsModule,
  ],
  templateUrl: './invoice-form.component.html',
})
export class InvoiceFormComponent implements OnInit {
  form!: FormGroup
  loading$ = new BehaviorSubject<boolean>(false)
  suggesting$ = new BehaviorSubject<boolean>(false)
  suggestions: string[] = []
  products: Product[] = []

  constructor(
    private fb: FormBuilder,
    private invoiceService: InvoiceService,
    private productService: ProductService,
    private router: Router,
    private snackBar: MatSnackBar,
  ) {}

  ngOnInit(): void {
    this.form = this.fb.group({
      description: [''],
      items: this.fb.array([]),
    })

    this.productService.list().subscribe((p) => (this.products = p))

    // AI suggestion trigger on description change
    this.form
      .get('description')!
      .valueChanges.pipe(debounceTime(600), distinctUntilChanged())
      .subscribe((val) => {
        if (val?.trim().length > 3) this.fetchSuggestions(val)
      })
  }

  get items(): FormArray {
    return this.form.get('items') as FormArray
  }

  addItem(): void {
    this.items.push(
      this.fb.group({
        product_id: [null, Validators.required],
        product_code: ['', Validators.required],
        description: ['', Validators.required],
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

  applySuggestion(suggestion: string): void {
    this.addItem()
    const last = this.items.length - 1
    this.items.at(last).patchValue({ description: suggestion })
  }

  private fetchSuggestions(description: string): void {
    this.suggesting$.next(true)
    this.invoiceService
      .suggest(description)
      .pipe(finalize(() => this.suggesting$.next(false)))
      .subscribe({
        next: (res) => (this.suggestions = res.suggestions),
        error: () => (this.suggestions = []),
      })
  }

  submit(): void {
    if (this.items.length === 0) {
      this.snackBar.open('Adicione ao menos um produto.', 'Fechar', {
        duration: 3000,
      })
      return
    }
    if (this.form.invalid) return

    this.loading$.next(true)
    this.invoiceService
      .create({ items: this.items.value })
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe({
        next: () => {
          this.snackBar.open('Nota fiscal criada.', 'Fechar', {
            duration: 3000,
          })
          this.router.navigate(['/invoices'])
        },
      })
  }
}

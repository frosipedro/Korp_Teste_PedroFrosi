import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import {
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  Validators,
} from '@angular/forms'
import { Router, ActivatedRoute } from '@angular/router'
import { MatFormFieldModule } from '@angular/material/form-field'
import { MatInputModule } from '@angular/material/input'
import { MatButtonModule } from '@angular/material/button'
import { MatIconModule } from '@angular/material/icon'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { BehaviorSubject, finalize } from 'rxjs'
import { ProductService } from '../../core/services/product.service'

@Component({
  selector: 'app-product-form',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  templateUrl: './product-form.component.html',
})
export class ProductFormComponent implements OnInit {
  form!: FormGroup
  loading$ = new BehaviorSubject<boolean>(false)
  editId: number | null = null

  get isEdit(): boolean {
    return this.editId !== null
  }

  constructor(
    private fb: FormBuilder,
    private productService: ProductService,
    private router: Router,
    private route: ActivatedRoute,
    private snackBar: MatSnackBar,
  ) {}

  ngOnInit(): void {
    this.form = this.fb.group({
      code: ['', [Validators.required, Validators.maxLength(50)]],
      description: ['', [Validators.required, Validators.maxLength(255)]],
      balance: [0, [Validators.required, Validators.min(0)]],
    })

    const id = this.route.snapshot.paramMap.get('id')
    if (id) {
      this.editId = +id
      this.loadProduct(this.editId)
    }
  }

  private loadProduct(id: number): void {
    this.loading$.next(true)
    this.productService
      .getById(id)
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe((p) => {
        this.form.patchValue(p)
        this.form.get('code')?.disable()
      })
  }

  submit(): void {
    if (this.form.invalid) return

    this.loading$.next(true)
    const value = this.form.getRawValue()

    const request$ = this.isEdit
      ? this.productService.update(this.editId!, {
          description: value.description,
          balance: value.balance,
        })
      : this.productService.create(value)

    request$.pipe(finalize(() => this.loading$.next(false))).subscribe({
      next: () => {
        this.snackBar.open(
          this.isEdit ? 'Produto atualizado.' : 'Produto criado.',
          'Fechar',
          { duration: 3000 },
        )
        this.router.navigate(['/products'])
      },
    })
  }
}

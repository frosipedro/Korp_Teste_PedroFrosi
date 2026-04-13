import { Component, OnInit } from '@angular/core'
import { CommonModule } from '@angular/common'
import { RouterLink } from '@angular/router'
import { MatTableModule } from '@angular/material/table'
import { MatButtonModule } from '@angular/material/button'
import { MatIconModule } from '@angular/material/icon'
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar'
import { MatChipsModule } from '@angular/material/chips'
import { BehaviorSubject, finalize } from 'rxjs'
import { ProductService } from '../../core/services/product.service'
import { Product } from '../../shared/models/product.model'

@Component({
  selector: 'app-product-list',
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
  ],
  templateUrl: './product-list.component.html',
})
export class ProductListComponent implements OnInit {
  products: Product[] = []
  loading$ = new BehaviorSubject<boolean>(false)
  displayedColumns = ['code', 'description', 'balance', 'actions']

  constructor(
    private productService: ProductService,
    private snackBar: MatSnackBar,
  ) {}

  ngOnInit(): void {
    this.load()
  }

  load(): void {
    this.loading$.next(true)
    this.productService
      .list()
      .pipe(finalize(() => this.loading$.next(false)))
      .subscribe((data) => (this.products = data))
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
}

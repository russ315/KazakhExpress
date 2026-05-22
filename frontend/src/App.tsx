import {
  ArrowRight,
  BadgeCheck,
  Boxes,
  Check,
  CreditCard,
  Heart,
  Minus,
  PackageCheck,
  Plus,
  Search,
  Send,
  ShieldCheck,
  ShoppingBag,
  Star,
  Truck,
  UserRound,
  WalletCards,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import './App.css'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

type Product = {
  id: string
  name: string
  description: string
  price_kzt: number
  stock: number
  image_url?: string
}

type User = {
  id: string
  email: string
  first_name: string
  last_name: string
}

type AuthResponse = {
  user: User
  token: string
  refresh_token: string
}

type CartItem = {
  product: Product
  quantity: number
}

type Order = {
  id: string
  customer_id: string
  status: string
  total_kzt: number
}

type Payment = {
  id: string
  order_id: string
  status: string
  amount_kzt: number
}

type Rating = {
  product_id: string
  average_rating: number
  review_count: number
}

type HealthState = {
  name: string
  path: string
  status: 'checking' | 'ok' | 'down'
}

type Notice = {
  tone: 'success' | 'error' | 'info'
  title: string
  body: string
}

const categories = ['All', 'Tech', 'Wearables', 'Travel', 'Home']

const money = new Intl.NumberFormat('kk-KZ', {
  style: 'currency',
  currency: 'KZT',
  maximumFractionDigits: 0,
})

function App() {
  const [route, setRoute] = useState(window.location.pathname)

  useEffect(() => {
    const onPop = () => setRoute(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  const navigate = (path: string) => {
    window.history.pushState({}, '', path)
    setRoute(path)
  }

  if (route.startsWith('/ops')) {
    return <OpsPage navigate={navigate} />
  }

  return <Storefront navigate={navigate} />
}

function Storefront({ navigate }: { navigate: (path: string) => void }) {
  const [products, setProducts] = useState<Product[]>([])
  const [loadingProducts, setLoadingProducts] = useState(true)
  const [ratings, setRatings] = useState<Record<string, Rating>>({})
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('All')
  const [cart, setCart] = useState<CartItem[]>([])
  const [user, setUser] = useState<User | null>(null)
  const [authOpen, setAuthOpen] = useState(false)
  const [checkoutOpen, setCheckoutOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)
  const [lastOrder, setLastOrder] = useState<Order | null>(null)
  const [lastPayment, setLastPayment] = useState<Payment | null>(null)

  const loadProducts = useCallback(async () => {
    setLoadingProducts(true)
    try {
      const list = await api<Product[]>('/products')
      setProducts(list)
      const ratingPairs = await Promise.all(
        list.map(async (product) => {
          try {
            return [product.id, await api<Rating>(`/products/${product.id}/rating`)] as const
          } catch {
            return [product.id, { product_id: product.id, average_rating: 0, review_count: 0 }] as const
          }
        }),
      )
      setRatings(Object.fromEntries(ratingPairs))
    } catch (error) {
      setNotice({
        tone: 'error',
        title: 'Catalog is unavailable',
        body: friendlyError(error),
      })
    } finally {
      setLoadingProducts(false)
    }
  }, [])

  useEffect(() => {
    void loadProducts()
  }, [loadProducts])

  const visibleProducts = useMemo(() => {
    const normalized = query.trim().toLowerCase()
    return products.filter((product) => {
      const matchesQuery =
        !normalized ||
        product.name.toLowerCase().includes(normalized) ||
        product.description.toLowerCase().includes(normalized)
      const matchesCategory =
        category === 'All' ||
        product.name.toLowerCase().includes(category.toLowerCase()) ||
        product.description.toLowerCase().includes(category.toLowerCase())
      return matchesQuery && matchesCategory
    })
  }, [category, products, query])

  const total = cart.reduce((sum, item) => sum + item.product.price_kzt * item.quantity, 0)
  const itemCount = cart.reduce((sum, item) => sum + item.quantity, 0)

  function addToCart(product: Product) {
    setCart((items) => {
      const existing = items.find((item) => item.product.id === product.id)
      if (existing) {
        return items.map((item) =>
          item.product.id === product.id
            ? { ...item, quantity: Math.min(item.quantity + 1, product.stock || item.quantity + 1) }
            : item,
        )
      }
      return [...items, { product, quantity: 1 }]
    })
    setCheckoutOpen(true)
  }

  function updateQuantity(productID: string, change: number) {
    setCart((items) =>
      items
        .map((item) =>
          item.product.id === productID
            ? { ...item, quantity: Math.max(0, item.quantity + change) }
            : item,
        )
        .filter((item) => item.quantity > 0),
    )
  }

  async function register(input: RegisterInput) {
    setBusy(true)
    try {
      const result = await api<AuthResponse>('/auth/register', {
        method: 'POST',
        body: JSON.stringify(input),
      })
      setUser(result.user)
      setAuthOpen(false)
      setNotice({
        tone: 'success',
        title: 'Account created',
        body: `Welcome email sent to ${result.user.email}.`,
      })
    } catch (error) {
      setNotice({ tone: 'error', title: 'Registration failed', body: friendlyError(error) })
    } finally {
      setBusy(false)
    }
  }

  async function checkout(form: CheckoutForm) {
    if (cart.length === 0) {
      setNotice({ tone: 'error', title: 'Cart is empty', body: 'Add at least one product before checkout.' })
      return
    }
    setBusy(true)
    try {
      const customerID = user?.id || 'guest-web'
      const customerEmail = user?.email || form.email
      const order = await api<Order>('/orders', {
        method: 'POST',
        body: JSON.stringify({
          customer_id: customerID,
          items: cart.map((item) => ({
            product_id: item.product.id,
            name: item.product.name,
            quantity: item.quantity,
            price_kzt: item.product.price_kzt,
          })),
        }),
      })
      const payment = await api<Payment>('/payment', {
        method: 'POST',
        body: JSON.stringify({
          order_id: order.id,
          customer_id: customerID,
          customer_email: customerEmail,
          amount_kzt: order.total_kzt,
          method: form.method,
          idempotency_key: `web-${order.id}`,
        }),
      })
      const settledOrder = await waitForPaidOrder(order.id)
      setLastOrder(settledOrder ?? order)
      setLastPayment(payment)
      setCart([])
      setNotice({
        tone: 'success',
        title: 'Payment accepted',
        body: `Order ${order.id} is ${settledOrder?.status ?? order.status}. Receipt email sent to ${customerEmail}.`,
      })
    } catch (error) {
      setNotice({ tone: 'error', title: 'Checkout failed', body: friendlyError(error) })
    } finally {
      setBusy(false)
    }
  }

  async function waitForPaidOrder(orderID: string) {
    for (let attempt = 0; attempt < 10; attempt += 1) {
      const order = await api<Order>(`/orders/${orderID}`)
      if (order.status === 'paid') {
        return order
      }
      await sleep(700)
    }
    return null
  }

  async function createReview(product: Product, rating: number, comment: string) {
    try {
      await api(`/products/${product.id}/reviews`, {
        method: 'POST',
        body: JSON.stringify({
          customer_id: user?.id || 'guest-review-web',
          rating,
          comment,
        }),
      })
      const nextRating = await api<Rating>(`/products/${product.id}/rating`)
      setRatings((items) => ({ ...items, [product.id]: nextRating }))
      setNotice({ tone: 'success', title: 'Review posted', body: `${product.name} rating updated.` })
    } catch (error) {
      setNotice({ tone: 'error', title: 'Review failed', body: friendlyError(error) })
    }
  }

  return (
    <div className="store-shell">
      <header className="store-header">
        <button className="brand-mark" type="button" onClick={() => navigate('/')}>
          <span className="brand-sigil">KZ</span>
          <span>
            <strong>KazakhExpress</strong>
            <small>Marketplace</small>
          </span>
        </button>

        <label className="search-box">
          <Search size={18} aria-hidden="true" />
          <span className="sr-only">Search products</span>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search phones, bags, headphones"
            type="search"
          />
        </label>

        <nav className="header-actions" aria-label="Store navigation">
          <a href="#catalog">Catalog</a>
          <a href="#journey">How it works</a>
          <button type="button" className="ghost-button" onClick={() => navigate('/ops')}>
            Ops
          </button>
          <button type="button" className="ghost-button" onClick={() => setAuthOpen(true)}>
            <UserRound size={17} aria-hidden="true" />
            {user ? user.first_name : 'Sign in'}
          </button>
          <button type="button" className="cart-button" onClick={() => setCheckoutOpen(true)}>
            <ShoppingBag size={18} aria-hidden="true" />
            Cart
            <span>{itemCount}</span>
          </button>
        </nav>
      </header>

      {notice && <NoticeBanner notice={notice} onClose={() => setNotice(null)} />}

      <main>
        <section className="consumer-hero">
          <div className="hero-copy">
            <h1>Shop the fastest KazakhExpress checkout flow.</h1>
            <p>
              Browse seeded products, create an order, pay with the mock provider, receive email,
              and leave a review without leaving the storefront.
            </p>
            <div className="hero-actions">
              <a className="primary-link" href="#catalog">
                Start shopping
                <ArrowRight size={18} aria-hidden="true" />
              </a>
              <button type="button" className="soft-button" onClick={() => setAuthOpen(true)}>
                Create account
              </button>
            </div>
          </div>
          <div className="hero-basket" aria-label="Checkout preview">
            <div className="journey-path" id="journey">
              {['Product', 'Cart', 'Order', 'Payment', 'Review'].map((step, index) => (
                <div className="journey-step" key={step}>
                  <span>{index + 1}</span>
                  <strong>{step}</strong>
                </div>
              ))}
            </div>
            <div className="hero-card">
              <PackageCheck size={28} aria-hidden="true" />
              <div>
                <strong>{lastOrder ? `Order ${lastOrder.status}` : 'Ready for checkout'}</strong>
                <span>
                  {lastPayment
                    ? `${money.format(lastPayment.amount_kzt)} paid`
                    : 'Payment receipt email is sent after checkout'}
                </span>
              </div>
            </div>
          </div>
        </section>

        <section className="market-layout" id="catalog">
          <aside className="filter-rail" aria-label="Catalog filters">
            <h2>Catalog</h2>
            <p>{loadingProducts ? 'Loading products' : `${visibleProducts.length} products available`}</p>
            <div className="category-list">
              {categories.map((item) => (
                <button
                  type="button"
                  key={item}
                  className={item === category ? 'selected' : ''}
                  onClick={() => setCategory(item)}
                >
                  {item}
                </button>
              ))}
            </div>
            <div className="trust-note">
              <ShieldCheck size={18} aria-hidden="true" />
              <span>Payments are idempotent, so repeated taps do not create duplicate charges.</span>
            </div>
          </aside>

          <div className="product-grid">
            {loadingProducts &&
              Array.from({ length: 6 }, (_, index) => (
                <div className="product-card product-skeleton" key={index} aria-hidden="true">
                  <div className="product-media" />
                  <div className="product-body">
                    <span />
                    <span />
                    <span />
                  </div>
                </div>
              ))}
            {visibleProducts.map((product) => (
              <ProductCard
                key={product.id}
                product={product}
                rating={ratings[product.id]}
                onAdd={() => addToCart(product)}
                onReview={(rating, comment) => createReview(product, rating, comment)}
              />
            ))}
          </div>
        </section>
      </main>

      {authOpen && (
        <AuthDialog
          busy={busy}
          onClose={() => setAuthOpen(false)}
          onSubmit={(input) => void register(input)}
        />
      )}

      <CheckoutDrawer
        cart={cart}
        open={checkoutOpen}
        busy={busy}
        user={user}
        total={total}
        onClose={() => setCheckoutOpen(false)}
        onQuantity={updateQuantity}
        onCheckout={(form) => void checkout(form)}
      />
    </div>
  )
}

function ProductCard({
  product,
  rating,
  onAdd,
  onReview,
}: {
  product: Product
  rating?: Rating
  onAdd: () => void
  onReview: (rating: number, comment: string) => void
}) {
  const [reviewOpen, setReviewOpen] = useState(false)
  const [stars, setStars] = useState(5)
  const [comment, setComment] = useState('Loved the checkout experience.')

  return (
    <article className="product-card">
      <div className="product-media">
        {product.image_url ? (
          <img src={product.image_url} alt={product.name} loading="lazy" />
        ) : (
          <Boxes size={44} aria-hidden="true" />
        )}
        <button type="button" aria-label={`Save ${product.name}`} className="wish-button">
          <Heart size={17} aria-hidden="true" />
        </button>
      </div>
      <div className="product-body">
        <div>
          <h3>{product.name}</h3>
          <p>{product.description}</p>
        </div>
        <div className="rating-line">
          <Star size={16} aria-hidden="true" />
          <span>{rating?.average_rating ? rating.average_rating.toFixed(1) : 'New'}</span>
          <small>{rating?.review_count ?? 0} reviews</small>
        </div>
        <div className="product-footer">
          <div>
            <strong>{money.format(product.price_kzt)}</strong>
            <small>{product.stock} in stock</small>
          </div>
          <button type="button" onClick={onAdd}>
            <Plus size={17} aria-hidden="true" />
            Add
          </button>
        </div>
        <button type="button" className="review-toggle" onClick={() => setReviewOpen((value) => !value)}>
          Leave a review
        </button>
        {reviewOpen && (
          <form
            className="review-form"
            onSubmit={(event) => {
              event.preventDefault()
              onReview(stars, comment)
              setReviewOpen(false)
            }}
          >
            <label>
              Rating
              <select value={stars} onChange={(event) => setStars(Number(event.target.value))}>
                {[5, 4, 3, 2, 1].map((value) => (
                  <option value={value} key={value}>
                    {value} stars
                  </option>
                ))}
              </select>
            </label>
            <label>
              Comment
              <input value={comment} onChange={(event) => setComment(event.target.value)} />
            </label>
            <button type="submit">Post review</button>
          </form>
        )}
      </div>
    </article>
  )
}

type RegisterInput = {
  email: string
  password: string
  first_name: string
  last_name: string
  phone: string
  address: string
}

function AuthDialog({
  busy,
  onClose,
  onSubmit,
}: {
  busy: boolean
  onClose: () => void
  onSubmit: (input: RegisterInput) => void
}) {
  const [input, setInput] = useState<RegisterInput>({
    email: 'demo@maqsatto.dev',
    password: 'Password123!',
    first_name: 'Aruzhan',
    last_name: 'Demo',
    phone: '+77010000000',
    address: 'Almaty, Dostyk Avenue 1',
  })

  return (
    <div className="modal-backdrop" role="presentation">
      <form
        className="auth-dialog"
        aria-label="Create account"
        onSubmit={(event) => {
          event.preventDefault()
          onSubmit(input)
        }}
      >
        <div className="dialog-head">
          <div>
            <h2>Create your account</h2>
            <p>Registration calls user-service and sends a welcome email through smtp-service.</p>
          </div>
          <button type="button" className="close-button" onClick={onClose}>
            Close
          </button>
        </div>
        <div className="form-grid">
          <TextInput label="Email" type="email" value={input.email} onChange={(email) => setInput({ ...input, email })} />
          <TextInput
            label="Password"
            type="password"
            value={input.password}
            onChange={(password) => setInput({ ...input, password })}
          />
          <TextInput
            label="First name"
            value={input.first_name}
            onChange={(first_name) => setInput({ ...input, first_name })}
          />
          <TextInput
            label="Last name"
            value={input.last_name}
            onChange={(last_name) => setInput({ ...input, last_name })}
          />
          <TextInput label="Phone" value={input.phone} onChange={(phone) => setInput({ ...input, phone })} />
          <TextInput label="Address" value={input.address} onChange={(address) => setInput({ ...input, address })} />
        </div>
        <button className="submit-button" type="submit" disabled={busy}>
          <Send size={17} aria-hidden="true" />
          {busy ? 'Creating account…' : 'Create account and send email'}
        </button>
      </form>
    </div>
  )
}

type CheckoutForm = {
  email: string
  method: 'card' | 'kaspi' | 'wallet'
}

function CheckoutDrawer({
  cart,
  open,
  busy,
  user,
  total,
  onClose,
  onQuantity,
  onCheckout,
}: {
  cart: CartItem[]
  open: boolean
  busy: boolean
  user: User | null
  total: number
  onClose: () => void
  onQuantity: (productID: string, change: number) => void
  onCheckout: (form: CheckoutForm) => void
}) {
  const [form, setForm] = useState<CheckoutForm>({
    email: user?.email ?? 'buyer@maqsatto.dev',
    method: 'card',
  })

  return (
    <aside className={open ? 'checkout-drawer open' : 'checkout-drawer'} aria-label="Cart and checkout">
      <div className="drawer-head">
        <div>
          <h2>Your cart</h2>
          <p>{cart.length === 0 ? 'Add products to start checkout.' : `${cart.length} product lines`}</p>
        </div>
        <button type="button" className="close-button" onClick={onClose}>
          Close
        </button>
      </div>

      <div className="cart-lines">
        {cart.map((item) => (
          <div className="cart-line" key={item.product.id}>
            <img src={item.product.image_url} alt="" />
            <div>
              <strong>{item.product.name}</strong>
              <span>{money.format(item.product.price_kzt)}</span>
            </div>
            <div className="quantity-control">
              <button type="button" onClick={() => onQuantity(item.product.id, -1)} aria-label="Decrease quantity">
                <Minus size={14} aria-hidden="true" />
              </button>
              <span>{item.quantity}</span>
              <button type="button" onClick={() => onQuantity(item.product.id, 1)} aria-label="Increase quantity">
                <Plus size={14} aria-hidden="true" />
              </button>
            </div>
          </div>
        ))}
      </div>

      <form
        className="checkout-form"
        onSubmit={(event) => {
          event.preventDefault()
          onCheckout(form)
        }}
      >
        <TextInput
          label="Receipt email"
          type="email"
          value={user?.email ?? form.email}
          onChange={(email) => setForm({ ...form, email })}
        />
        <label className="field">
          Payment method
          <div className="payment-options">
            {[
              ['card', CreditCard, 'Card'],
              ['kaspi', WalletCards, 'Kaspi'],
              ['wallet', ShoppingBag, 'Wallet'],
            ].map(([value, Icon, label]) => (
              <button
                type="button"
                key={value as string}
                className={form.method === value ? 'selected' : ''}
                onClick={() => setForm({ ...form, method: value as CheckoutForm['method'] })}
              >
                <Icon size={17} aria-hidden="true" />
                {label as string}
              </button>
            ))}
          </div>
        </label>
        <div className="checkout-total">
          <span>Total</span>
          <strong>{money.format(total)}</strong>
        </div>
        <button className="submit-button" type="submit" disabled={busy || cart.length === 0}>
          <BadgeCheck size={18} aria-hidden="true" />
          {busy ? 'Processing payment…' : 'Place order and pay'}
        </button>
      </form>
    </aside>
  )
}

function OpsPage({ navigate }: { navigate: (path: string) => void }) {
  const [health, setHealth] = useState<HealthState[]>([
    { name: 'Gateway', path: '/health', status: 'checking' },
    { name: 'Products', path: '/products/health', status: 'checking' },
    { name: 'Orders', path: '/orders/health', status: 'checking' },
    { name: 'Payment', path: '/payment/health', status: 'checking' },
    { name: 'Reviews', path: '/reviews/health', status: 'checking' },
  ])

  useEffect(() => {
    void Promise.all(
      health.map(async (item) => {
        try {
          await api(item.path)
          setHealth((current) =>
            current.map((row) => (row.path === item.path ? { ...row, status: 'ok' } : row)),
          )
        } catch {
          setHealth((current) =>
            current.map((row) => (row.path === item.path ? { ...row, status: 'down' } : row)),
          )
        }
      }),
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <main className="ops-page">
      <button type="button" className="soft-button" onClick={() => navigate('/')}>
        Back to store
      </button>
      <h1>Ops Route</h1>
      <p>Separate backend visibility for local demos. The storefront stays consumer-first.</p>
      <div className="ops-grid">
        {health.map((item) => (
          <div className="ops-card" key={item.path}>
            <span className={item.status}>
              {item.status === 'ok' ? <Check size={16} aria-hidden="true" /> : <Truck size={16} aria-hidden="true" />}
              {item.status}
            </span>
            <strong>{item.name}</strong>
            <small>{API_BASE_URL + item.path}</small>
          </div>
        ))}
      </div>
    </main>
  )
}

function TextInput({
  label,
  value,
  onChange,
  type = 'text',
}: {
  label: string
  value: string
  onChange: (value: string) => void
  type?: string
}) {
  return (
    <label className="field">
      {label}
      <input type={type} value={value} onChange={(event) => onChange(event.target.value)} required />
    </label>
  )
}

function NoticeBanner({ notice, onClose }: { notice: Notice; onClose: () => void }) {
  return (
    <div className={`notice ${notice.tone}`} role="status">
      <div>
        <strong>{notice.title}</strong>
        <span>{notice.body}</span>
      </div>
      <button type="button" onClick={onClose}>
        Dismiss
      </button>
    </div>
  )
}

async function api<T = unknown>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init.headers,
    },
  })
  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || response.statusText)
  }
  return response.json() as Promise<T>
}

function friendlyError(error: unknown) {
  if (error instanceof Error) {
    return error.message.replaceAll('"', '')
  }
  return 'Try again in a moment.'
}

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

export default App

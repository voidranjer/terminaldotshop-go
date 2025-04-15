# Code Quality Improvement TODO

This document outlines code quality issues in priority order with specific implementation guidance for future Claude instances.

## 1. Standardize Error Handling

**Impact**: High - Improves reliability and debuggability

**Issues**:
- Inconsistent error handling patterns
- Silently ignored errors (e.g., in cart.go)
- Panics in initialization code (resource.go)
- Missing context in error returns

**Implementation Plan**:
1. Create a central error handling package at `pkg/errors` with:
   - Custom error types for different domains (API, UI, Resource)
   - Context-preserving error wrapping functions
   - Standardized error formatting

2. Replace direct error returns with wrapped errors:
   ```go
   // Before
   return nil, err
   
   // After
   return nil, errors.Wrap(err, "failed to fetch product data")
   ```

3. Fix silent error ignores, especially in cart.go line 122 where errors are commented out.

4. Convert panics to proper error returns in resource.go initialization:
   ```go
   // Before
   if err != nil {
       panic(fmt.Sprintf("failed to load resource: %s", err))
   }
   
   // After
   if err != nil {
       return nil, fmt.Errorf("failed to load resource: %w", err)
   }
   ```

5. Add error logging to appropriate places with context using a structured logger.

## 2. Improve Test Coverage

**Impact**: High - Ensures functionality works as expected and prevents regressions

**Issues**:
- Minimal test coverage across packages
- Placeholder tests that don't verify functionality
- No tests for core business logic

**Implementation Plan**:
1. Create test helpers at `pkg/testutil` with:
   - Mock API responses
   - Test fixtures
   - UI component test utilities

2. Expand API tests to cover all endpoints with proper assertions:
   ```go
   func TestGetProducts(t *testing.T) {
       client := api.NewClient("test-token")
       client.HTTPClient = &mockHTTPClient{responseJSON: `{"products":[...]}`}
       
       products, err := client.GetProducts()
       
       assert.NoError(t, err)
       assert.Equal(t, 5, len(products))
       assert.Equal(t, "Test Product", products[0].Name)
   }
   ```

3. Add unit tests for utility functions in validate package.

4. Create component tests for bubble tea models:
   ```go
   func TestCartUpdate(t *testing.T) {
       m := tui.NewCart()
       m.Products = []api.Product{testProduct}
       
       newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
       
       cartModel, ok := newModel.(tui.Cart)
       assert.True(t, ok)
       assert.NotNil(t, cmd)
       assert.Equal(t, 1, len(cartModel.SelectedItems))
   }
   ```

5. Set up CI test coverage reporting with a target of at least 70% coverage.

## 3. Refactor Model Architecture

**Impact**: High - Improves maintainability and testability

**Issues**:
- Monolithic model struct in root.go
- No clear separation between UI state, business logic, and API
- Strong coupling between components

**Implementation Plan**:
1. Split the monolithic model in root.go into domain-specific models:
   - UserModel (auth, account info)
   - APIClient (API connection handling)
   - ShopModel (products, cart)
   - UIStateModel (navigation, screen state)

2. Create interfaces for better dependency injection:
   ```go
   type APIService interface {
       GetProducts() ([]Product, error)
       AddToCart(productID string, quantity int) error
       // other API methods
   }
   
   // Then use this interface in models
   type ShopModel struct {
       api APIService
       // other fields
   }
   ```

3. Implement a state management pattern for better data flow between components.

4. Create a routing system to manage screen transitions cleanly.

5. Use dependency injection for API client and other services in models.

## 4. Standardize Return Patterns

**Impact**: Medium-High - Makes code more predictable and easier to use correctly

**Issues**:
- Inconsistent return values across similar functions
- Mixed return patterns (errors vs modified models)
- Unexplained nil returns in error cases

**Implementation Plan**:
1. Create standard result types for different operations:
   ```go
   type APIResult[T any] struct {
       Data  T
       Error error
   }
   
   type UIActionResult struct {
       Model tea.Model
       Cmd   tea.Cmd
       Error error
   }
   ```

2. Update function signatures to follow consistent patterns:
   - For operations: `func DoSomething() (result, error)`
   - For Bubble Tea updates: `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`

3. Document expected return values in function comments.

4. Replace unexplained nil returns with proper error messages.

5. Create helper functions for common return patterns.

## 5. Add Documentation

**Impact**: Medium - Improves developer experience and onboarding

**Issues**:
- Most public functions lack documentation
- Complex logic without explanatory comments
- Missing package-level documentation

**Implementation Plan**:
1. Add package-level documentation to each package:
   ```go
   // Package api provides client interfaces for interacting with the Terminal Shop API.
   // It handles authentication, request formatting, and response parsing.
   package api
   ```

2. Document all exported functions, types, and constants:
   ```go
   // GetProducts fetches the complete product catalog from the API.
   // It returns a slice of Product objects or an error if the request fails.
   // Results are sorted by popularity by default.
   func (c *Client) GetProducts() ([]Product, error) {
   ```

3. Add detailed comments for complex algorithms:
   ```go
   // CcnValidator implements the Luhn algorithm to validate credit card numbers:
   // 1. Starting from the rightmost digit, double every second digit
   // 2. If doubling results in a two-digit number, add those digits together
   // 3. Sum all digits in the resulting sequence
   // 4. If the sum is divisible by 10, the number is valid
   ```

4. Create examples for key functions.

5. Add architecture diagrams in README.md explaining component relationships.

## 6. Extract UI Constants and Improve Layout

**Impact**: Medium - Makes UI more maintainable and adaptable

**Issues**:
- Hard-coded UI dimensions and spacing
- Difficult to adapt to different terminal sizes
- Inconsistent styling

**Implementation Plan**:
1. Create a `pkg/tui/constants.go` file with UI constants:
   ```go
   package tui

   // Layout constants
   const (
       DefaultMargin      = 1
       DefaultPadding     = 2
       DefaultWidth       = 80
       HeaderHeight       = 3
       FooterHeight       = 2
       ContentMinHeight   = 10
   )

   // Color constants (matching theme)
   var (
       PrimaryColor   = lipgloss.Color("#1D76DB")
       SecondaryColor = lipgloss.Color("#FFDB58")
       ErrorColor     = lipgloss.Color("#E84855")
       TextColor      = lipgloss.Color("#333333")
   )
   ```

2. Create responsive layout functions:
   ```go
   func ResponsiveWidth(containerWidth int) int {
       if containerWidth < 60 {
           return containerWidth - 2*DefaultMargin
       }
       return DefaultWidth
   }
   ```

3. Replace hard-coded values with constants and layout functions.

4. Create UI helper components for common patterns (cards, lists, forms).

5. Implement proper window size handling for resize events.

## 7. Improve Resource Management

**Impact**: Medium - Prevents resource leaks

**Issues**:
- API connections aren't properly closed
- No clear lifecycle management
- Resource cleanup inconsistencies

**Implementation Plan**:
1. Implement proper close methods:
   ```go
   type APIClient struct {
       httpClient *http.Client
       // other fields
   }
   
   func (c *APIClient) Close() error {
       // Close idle connections
       c.httpClient.CloseIdleConnections()
       return nil
   }
   ```

2. Use defer for resource cleanup:
   ```go
   func processRequest() error {
       client, err := api.NewClient()
       if err != nil {
           return err
       }
       defer client.Close()
       
       // Use client...
   }
   ```

3. Create a resource manager to track and clean up resources:
   ```go
   type ResourceManager struct {
       resources []io.Closer
       mu        sync.Mutex
   }
   
   func (rm *ResourceManager) Register(r io.Closer) {
       rm.mu.Lock()
       defer rm.mu.Unlock()
       rm.resources = append(rm.resources, r)
   }
   
   func (rm *ResourceManager) CloseAll() error {
       // Close all registered resources
   }
   ```

4. Add shutdown hooks to ensure clean program termination.

5. Add context support for cancellation propagation.

## 8. Clean Up Commented Code

**Impact**: Low-Medium - Improves readability and maintenance

**Issues**:
- Several instances of commented-out code
- Suggests incomplete refactoring
- Creates confusion about intended implementation

**Implementation Plan**:
1. Systematically review and remove all commented-out code, focusing first on:
   - Error handling comments in cart.go
   - Debug print statements
   - Alternative implementations left as comments

2. For code that might be needed later, extract to utility functions instead of commenting out.

3. Add TODOs with specific requirements for incomplete features:
   ```go
   // TODO(username): Implement offline mode support that caches products
   // and syncs when connection is restored
   ```

4. Create proper feature flags for experimental features rather than using comments.

5. Document architectural decisions in separate markdown files rather than code comments.
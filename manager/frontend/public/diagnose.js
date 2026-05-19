// Script chẩn đoán frontend
console.log('=== Bắt đầu chẩn đoán frontend ===')

// Kiểm tra môi trường cơ bản
console.log('1. Kiểm tra môi trường cơ bản:')
console.log('   - Phiên bản Vue:', typeof window.Vue !== 'undefined' ? 'Đã tải Vue' : 'Chưa tải Vue')
console.log('   - URL hiện tại:', window.location.href)
console.log('   - User Agent:', navigator.userAgent)

// Kiểm tra localStorage
console.log('2. Kiểm tra bộ nhớ cục bộ:')
console.log('   - Token:', localStorage.getItem('token'))
console.log('   - User:', localStorage.getItem('user'))

// Kiểm tra kết nối mạng
console.log('3. Kiểm tra kết nối backend:')
fetch('http://localhost:8080/api/profile')
  .then(response => {
    console.log('   - Trạng thái phản hồi backend:', response.status)
    if (response.status === 401) {
      console.log('   - Backend đang chạy bình thường (trả về lỗi 401 chưa xác thực)')
    }
  })
  .catch(error => {
    console.log('   - Kết nối backend thất bại:', error.message)
  })

// Kiểm tra route
console.log('4. Các route kiểm thử khả dụng:')
console.log('   - /test - Trang kiểm thử cơ bản')
console.log('   - /simple-login - Trang đăng nhập rút gọn')
console.log('   - /login - Trang đăng nhập đầy đủ')

console.log('=== Kết thúc chẩn đoán frontend ===')
console.log('Vui lòng xem thông tin trên trong console trình duyệt')

// Xuất ra global để tiện gọi từ console
window.diagnose = () => {
  console.clear()
  // Chạy lại chẩn đoán
  location.reload()
}

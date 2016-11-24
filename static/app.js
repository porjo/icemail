var apiURL = '//localhost:8080/api';

var app = new Vue({
	el: '#app',
	data: {
		headers: null
	},

	created: function () {
		this.fetchData()
	},

	methods: {
		fetchData: function () {
			$.post(apiURL + '/list', '{}', function(data) {
				console.log(data);
				self.headers = data;
			});
		}
	}
})

var apiURL = '//localhost:8080/api';

$(function() {
	var app = new Vue({
		el: '#app',
		data: {
			headers: null,
		},

		created: function () {
			this.fetchData()
		},

		methods: {
			fetchData: function () {
				var self = this;
				$.post(apiURL + '/list', '{}', function(data) {
					self.headers = data;
				});
			}
		}
	});
});

var apiURL = '//localhost:8080/api';

var fields = [
	"From",
	"To",
	"Subject"
];

$(function() {
	var app = new Vue({
		el: '#app',
		data: {
			query: "",
			result: {
				emails: [],
				total: 0
			},
			fields: fields,
			searchFields: fields,
			searchDays: 0
		},

		created: function () {
			this.fetchData()
		},

		methods: {
			fetchData: function () {
				var self = this;
				$.post(apiURL + '/list', '{}', function(data) {
					cleanData(data);
					self.result.emails = data.Emails;
					self.result.total = data.Total
				});
			},

			toggleSearchOptions: function() {
				$('#search_options').slideToggle();
			},

			search: function() {
				if (this.query == ''){
					this.fetchData();
					return;
				}
				var self = this;

				var data = {
					query: this.query,
					locations: this.searchFields,
				};

				if( this.searchDays > 0) {
					var startTime = new Date();
					startTime.setDate(startTime.getDate() - this.searchDays);
					data.starttime = startTime.toISOString();
				}

				$.ajax({
					url: apiURL + '/search',
					type: 'post',
					data: JSON.stringify(data),
					contentType: 'application/json',
					dataType: 'json',
					success: function(data) {
						cleanData(data);
						self.result.emails = data.Emails;
						self.result.total = data.Total
					}
				});
			}
		}
	});
});

function cleanData(data) {
	$.each(data.emails, function(k,v) {
		if( typeof v.Header.Date === 'undefined' )
			v.Header.Date = '';
		if( typeof v.Header.From === 'undefined' )
			v.Header.From = '';
		if( typeof v.Header.To === 'undefined' )
			v.Header.To = '';
	});
}

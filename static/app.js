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
			searchDays: 0,
			showModal: false,
			modal: {
				Title: '',
				Body: ''
			}
		},

		created: function () {
			this.fetchData()
		},

		methods: {

			closeModal: function() {
				$("body").removeClass('modal-open');
				this.showModal = false;
			},

			viewMessage: function(id) {
				var self = this;
				$("body").addClass('modal-open');
				$.get(apiURL + '/search/' + id, function(data) {
					self.modal.Title = data.Emails[0].Header.Subject[0];
					self.modal.Body = data.Emails[0].Body;
					self.showModal = true;
				});
			},

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

	Vue.component('modal', {
		template: '#modal-template'
	})

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
});

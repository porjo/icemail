$(function() {
	const apiURL = '//localhost:8080/api';

	const fields = [
		"From",
		"To",
		"Subject"
	];

	var store = {
		state: {
			limit: 20,
			searchDays: 0,
			fields: fields
		}
	};

	const dateFormat = "ddd, DD MMM YYYY HH:mm:ss Z"

	Vue.filter('fromNow', function(value) {
		if( moment.isMoment(value) ) {
			return value.fromNow();
		} else {
			return moment(value, dateFormat).fromNow();
		}
	});

	Vue.filter('formatted', function(value) {
		if( moment.isMoment(value) ) {
			return value.format(dateFormat);
		} else {
			return moment(value, dateFormat).format();
		}
	});

	Vue.filter('commaList', function(list) {
		if( !$.isArray(list) ) { return ''; }

		var str = '';
		$.each(list, function(k,v) {
			str += v + ', ';
		});
		return str.replace(/, $/,'');
	});

	const Message = Vue.extend({
		template: '#message-view',
		data: function() {
			return {
				header: {},
				body: '',
				id: 0,
				delivered: '',
				error: '',
			}
		},

		watch: {
			'$route': 'viewMsg'
		},

		created: function () {
			this.viewMsg();
		},

		methods: {
			goBack: function() {
				router.go(-1);
			},

			resetError: function() {
				this.error = '';
			},

			sendMsg: function(id) {
				var self = this;
				$.get(apiURL + '/mail/' + id, function(data) {
					if('Success' in data) {
						if(data.Success) {
							self.delivered = moment()
						}
					}
				}).fail( function(xhr, ajaxOptions, thrownError) {
					self.error = xhr.responseText;
				});
			},

			viewMsg: function() {
				var self = this;
				$.get(apiURL + '/search/' + this.$route.params.id, function(data) {
					self.header = data.Emails[0].Header;
					self.body = data.Emails[0].Body;
					self.id = data.Emails[0].ID;
					if( 'Delivered' in data.Emails[0] ) {
						self.delivered = moment(data.Emails[0].Delivered);
					} else {
						self.delivered = '';
					}
				}).fail( function(xhr, ajaxOptions, thrownError) {
					self.error = xhr.responseText;
				});
			}
		}
	});

	const List = Vue.extend({
		template: '#message-list',

		data: function() {
			return {
				request: {
					query: "",
					limit: 0,
					offset: 0,
					locations: []
				},
				result: {
					emails: [],
					total: 0,
					offset: 0,
					error: ''
				},
				fields: fields,
				state: store.state,
			}
		},

		watch: {
			'$route' (to, from) {
				if(to.fullPath != from.fullPath) {
					this.searchMsg();
				}
			}
		},

		created: function () {
			if(this.$route.query.query != '') {
				this.request.query = this.$route.query.query;
			}

			this.searchMsg();
		},

		methods: {

			resetError: function() {
				this.result.error = '';
			},

			currentPage: function() {
				return Math.ceil(this.result.offset / this.state.limit) + 1;
			},

			// used to set active CSS class
			pageActive: function(n) {
				var page = this.currentPage();
				var val = (n == page);
				return {active: val}
			},

			viewMsg: function(id) {
				// ignore click if mouse selected
				var haveSel = getSelection().toString().length > 0;
				if( haveSel ) {
					return;
				}
				router.push({ name: 'message', params: { id: id}});
			},

			toggleSearchOptions: function() {
				$('#search_options').slideToggle();
			},

			searchQuery: function() {
				if(this.request.query == this.$route.query.query ||
					(this.request.query == '' && typeof this.$route.query.query == "undefined") ) {
					this.searchMsg();
				} else if(this.request.query != '') {
					router.push({ name: 'search', query: {query: this.request.query}});
				} else {
					router.push({ name: 'search', query: {}});
				}
			},

			searchMsg: function() {

				var request = $.extend({}, this.request);
				request.limit = this.state.limit;
				request.locations = this.state.fields;

				this.resetError();

				if( this.state.searchDays > 0) {
					var startTime = new Date();
					startTime.setDate(startTime.getDate() - this.state.searchDays);
					request.starttime = startTime.toISOString();
				}

				if( 'page' in this.$route.query ) {
					request.offset = request.limit * (this.$route.query.page-1);
				}

				var self = this;
				$.ajax({
					url: apiURL + '/search',
					type: 'post',
					data: JSON.stringify(request),
					contentType: 'application/json',
					dataType: 'json',
					success: function(data) {
						cleanData(data);
						self.result.emails = data.Emails;
						self.result.total = data.Total;
						self.result.offset = data.Offset;
						self.result.pages = Math.ceil( data.Total / request.limit );
					},
					error: function (xhr, ajaxOptions, thrownError) {
						self.result.error = xhr.responseText;
					}
				});
			}
		}
	});

	const routes = [
		{ name: 'search', path: '/', component: List },
		{ name: 'message', path: '/message/:id', component: Message }
	];

	const router = new VueRouter({
		routes: routes,
	});

	const App = new Vue({
		router,
	}).$mount('#app');

	function cleanData(data) {
		$.each(data.Emails, function(k,v) {
			if( typeof v.Header.Date === 'undefined' )
				v.Header.Date = '';
			if( typeof v.Header.From === 'undefined' )
				v.Header.From = '';
			if( typeof v.Header.To === 'undefined' )
				v.Header.To = '';
		});
	}
});
